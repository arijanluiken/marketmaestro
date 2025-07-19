package order

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

// Enhanced order types
const (
	OrderTypeMarket     = "market"
	OrderTypeLimit      = "limit"
	OrderTypeStopMarket = "stop_market"
	OrderTypeStopLimit  = "stop_limit"
	OrderTypeTrailing   = "trailing_stop"
)

// Order statuses
const (
	StatusPending         = "pending"
	StatusOpen            = "open"
	StatusPartiallyFilled = "partially_filled"
	StatusFilled          = "filled"
	StatusCancelled       = "cancelled"
	StatusRejected        = "rejected"
)

// Messages for order manager actor communication
type (
	PlaceOrderMsg struct {
		Symbol       string
		Side         string // "buy" or "sell"
		Type         string // OrderType constants
		Quantity     float64
		Price        float64
		StopPrice    float64 // For stop orders
		TrailAmount  float64 // For trailing stops (absolute)
		TrailPercent float64 // For trailing stops (percentage)
		TimeInForce  string  // "GTC", "IOC", "FOK"
		Reason       string
	}

	PlaceTrailingStopMsg struct {
		Symbol       string
		Side         string
		Quantity     float64
		TrailAmount  float64 // Absolute trail amount
		TrailPercent float64 // Percentage trail amount
		Reason       string
	}

	PlaceStopOrderMsg struct {
		Symbol     string
		Side       string
		Quantity   float64
		StopPrice  float64
		LimitPrice float64 // Optional, for stop-limit orders
		Reason     string
	}

	CancelOrderMsg struct {
		OrderID string
		Symbol  string
	}

	ModifyOrderMsg struct {
		OrderID      string
		Symbol       string
		NewQuantity  *float64
		NewPrice     *float64
		NewStopPrice *float64
	}

	GetOrdersMsg   struct{ Symbol string }
	OrderUpdateMsg struct{ Order *EnhancedOrder }
	StatusMsg      struct{}
	PriceUpdateMsg struct {
		Symbol string
		Price  float64
	}
)

// EnhancedOrder extends the basic Order with advanced features
type EnhancedOrder struct {
	*exchanges.Order
	OriginalType  string  // Original order type before conversion
	StopPrice     float64 // Stop trigger price
	TrailAmount   float64 // Trailing stop amount (absolute)
	TrailPercent  float64 // Trailing stop percentage
	HighWaterMark float64 // For trailing stops (highest price for sell, lowest for buy)
	TimeInForce   string  // "GTC", "IOC", "FOK"
	CreatedAt     time.Time
	UpdatedAt     time.Time
	TriggerPrice  float64 // Last trigger price for stop orders
	IsTriggered   bool    // Whether stop order has been triggered
	ParentOrderID string  // For stop orders created from other orders
}

// OrderManagerActor manages order placement and advanced order types
type OrderManagerActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
	orders       map[string]*EnhancedOrder // Active orders by ID
	exchange     exchanges.Exchange        // Reference to exchange interface

	// Advanced order management
	stopOrders    map[string]*EnhancedOrder // Stop orders waiting for trigger
	trailingStops map[string]*EnhancedOrder // Trailing stop orders
	priceCache    map[string]float64        // Latest prices by symbol
	mutex         sync.RWMutex              // Thread safety

	// Monitoring
	tickerTimer    *time.Ticker
	monitoringDone chan struct{}
}

// New creates a new order manager actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *OrderManagerActor {
	return &OrderManagerActor{
		exchangeName:   exchangeName,
		config:         cfg,
		db:             db,
		logger:         logger,
		orders:         make(map[string]*EnhancedOrder),
		stopOrders:     make(map[string]*EnhancedOrder),
		trailingStops:  make(map[string]*EnhancedOrder),
		priceCache:     make(map[string]float64),
		monitoringDone: make(chan struct{}),
	}
}

// SetExchange sets the exchange interface for order operations
func (o *OrderManagerActor) SetExchange(exchange exchanges.Exchange) {
	o.exchange = exchange

	// Trigger order sync now that we have an exchange interface
	go o.syncOrdersFromExchange(nil)
}

// Receive handles incoming messages
func (o *OrderManagerActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		o.onStarted(ctx)
	case actor.Stopped:
		o.onStopped(ctx)
	case actor.Initialized:
		o.onInitialized(ctx)
	case PlaceOrderMsg:
		o.onPlaceOrder(ctx, msg)
	case PlaceTrailingStopMsg:
		o.onPlaceTrailingStop(ctx, msg)
	case PlaceStopOrderMsg:
		o.onPlaceStopOrder(ctx, msg)
	case CancelOrderMsg:
		o.onCancelOrder(ctx, msg)
	case ModifyOrderMsg:
		o.onModifyOrder(ctx, msg)
	case GetOrdersMsg:
		o.onGetOrders(ctx, msg)
	case OrderUpdateMsg:
		o.onOrderUpdate(ctx, msg)
	case PriceUpdateMsg:
		o.onPriceUpdate(ctx, msg)
	case StatusMsg:
		o.onStatus(ctx)
	case map[string]interface{}: // Handle strategy signals
		o.onStrategySignal(ctx, msg)
	default:
		// Reduced chattiness - only log unknown message types at info level
		if fmt.Sprintf("%T", msg) != "map[string]interface {}" {
			o.logger.Info().
				Str("message_type", fmt.Sprintf("%T", msg)).
				Msg("Received unknown message type")
		}
	}
}

func (o *OrderManagerActor) onStarted(ctx *actor.Context) {
	o.logger.Info().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor started")

	// Start price monitoring for stop and trailing orders
	o.startPriceMonitoring(ctx)
}

func (o *OrderManagerActor) onInitialized(ctx *actor.Context) {
	o.logger.Debug().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor initialized")

	// Order sync will be triggered when exchange interface is set
}

// syncOrdersFromExchange synchronizes orders from the exchange on startup
func (o *OrderManagerActor) syncOrdersFromExchange(ctx *actor.Context) {
	if o.exchange == nil {
		o.logger.Warn().Msg("No exchange interface available for order sync")
		return
	}

	o.logger.Info().Msg("Starting order synchronization from exchange")

	// Get all open orders from exchange (empty symbol gets all)
	exchangeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exchangeOrders, err := o.exchange.GetOpenOrders(exchangeCtx, "")
	if err != nil {
		o.logger.Error().Err(err).Msg("Failed to get open orders from exchange")
		return
	}

	// Get existing orders from database
	existingOrders, err := o.db.GetAllOpenOrders()
	if err != nil {
		o.logger.Error().Err(err).Msg("Failed to get orders from database")
		return
	}

	// Create maps for efficient lookup
	exchangeOrderMap := make(map[string]*exchanges.Order)
	for _, order := range exchangeOrders {
		exchangeOrderMap[order.ID] = order
	}

	existingOrderMap := make(map[string]*database.Order)
	for _, order := range existingOrders {
		existingOrderMap[order.ExchangeOrderID] = order
	}

	// Sync orders
	var syncedCount, cancelledCount, newCount int

	// 1. Update existing orders and mark cancelled ones
	for _, dbOrder := range existingOrders {
		if exchangeOrder, exists := exchangeOrderMap[dbOrder.ExchangeOrderID]; exists {
			// Order still exists on exchange, update status if needed
			if o.needsStatusUpdate(dbOrder, exchangeOrder) {
				err := o.updateOrderFromExchange(dbOrder, exchangeOrder)
				if err != nil {
					o.logger.Error().Err(err).
						Int64("order_id", dbOrder.ID).
						Msg("Failed to update order from exchange")
				} else {
					syncedCount++
				}
			}
		} else if dbOrder.Status == StatusOpen || dbOrder.Status == StatusPartiallyFilled {
			// Order doesn't exist on exchange but is open in DB - mark as cancelled
			err := o.db.UpdateOrderStatus(dbOrder.ExchangeOrderID, StatusCancelled)
			if err != nil {
				o.logger.Error().Err(err).
					Int64("order_id", dbOrder.ID).
					Msg("Failed to update cancelled order status")
			} else {
				cancelledCount++
				o.logger.Info().
					Int64("order_id", dbOrder.ID).
					Str("exchange_order_id", dbOrder.ExchangeOrderID).
					Msg("Marked order as cancelled (not found on exchange)")
			}
		}
	}

	// 2. Add new orders found on exchange but not in database
	for _, exchangeOrder := range exchangeOrders {
		if _, exists := existingOrderMap[exchangeOrder.ID]; !exists {
			// This is a new order not in our database
			dbOrder := &database.Order{
				ExchangeOrderID: exchangeOrder.ID,
				Symbol:          exchangeOrder.Symbol,
				Side:            exchangeOrder.Side,
				Type:            exchangeOrder.Type,
				Quantity:        exchangeOrder.Quantity,
				Price:           exchangeOrder.Price,
				Status:          exchangeOrder.Status,
				CreatedAt:       exchangeOrder.Time,
				UpdatedAt:       time.Now(),
				Exchange:        o.exchangeName,
			}

			err := o.db.SaveOrder(dbOrder)
			if err != nil {
				o.logger.Error().Err(err).
					Str("exchange_order_id", exchangeOrder.ID).
					Msg("Failed to save synced order")
			} else {
				newCount++
				o.logger.Info().
					Int64("order_id", dbOrder.ID).
					Str("exchange_order_id", exchangeOrder.ID).
					Str("symbol", exchangeOrder.Symbol).
					Msg("Added order from exchange sync")
			}
		}
	}

	o.logger.Info().
		Int("synced", syncedCount).
		Int("cancelled", cancelledCount).
		Int("new", newCount).
		Int("total_exchange_orders", len(exchangeOrders)).
		Msg("Order synchronization completed")
}

// needsStatusUpdate checks if the database order needs status update from exchange
func (o *OrderManagerActor) needsStatusUpdate(dbOrder *database.Order, exchangeOrder *exchanges.Order) bool {
	return dbOrder.Status != exchangeOrder.Status
}

// updateOrderFromExchange updates database order with exchange order data
func (o *OrderManagerActor) updateOrderFromExchange(dbOrder *database.Order, exchangeOrder *exchanges.Order) error {
	dbOrder.Status = exchangeOrder.Status
	dbOrder.UpdatedAt = time.Now()

	return o.db.UpdateOrder(dbOrder)
}

func (o *OrderManagerActor) onStopped(ctx *actor.Context) {
	o.logger.Info().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor stopped")

	// Stop price monitoring
	o.stopPriceMonitoring()
}

func (o *OrderManagerActor) startPriceMonitoring(ctx *actor.Context) {
	o.tickerTimer = time.NewTicker(1 * time.Second) // Check every second

	go func() {
		for {
			select {
			case <-o.tickerTimer.C:
				o.checkStopOrders(ctx)
				o.updateTrailingStops(ctx)
			case <-o.monitoringDone:
				return
			}
		}
	}()
}

func (o *OrderManagerActor) stopPriceMonitoring() {
	if o.tickerTimer != nil {
		o.tickerTimer.Stop()
	}
	close(o.monitoringDone)
}

func (o *OrderManagerActor) onStrategySignal(ctx *actor.Context, signal map[string]interface{}) {
	// Check if this is a set_exchange message
	if action, ok := signal["action"].(string); ok && action == "set_exchange" {
		if exchange, ok := signal["exchange"].(exchanges.Exchange); ok {
			o.SetExchange(exchange)
			o.logger.Info().Msg("Exchange interface set")
			return
		}
	}

	// Check if this is a price update
	if msgType, ok := signal["type"].(string); ok && msgType == "price_update" {
		if symbol, ok := signal["symbol"].(string); ok {
			if price, ok := signal["price"].(float64); ok {
				o.onPriceUpdate(ctx, PriceUpdateMsg{Symbol: symbol, Price: price})
				return
			}
		}
	}

	// Convert strategy signal to order message
	symbol, _ := signal["symbol"].(string)
	side, _ := signal["side"].(string)
	orderType, _ := signal["type"].(string)
	quantity, _ := signal["quantity"].(float64)
	price, _ := signal["price"].(float64)
	reason, _ := signal["reason"].(string)

	// Advanced order parameters
	stopPrice, _ := signal["stop_price"].(float64)
	trailAmount, _ := signal["trail_amount"].(float64)
	trailPercent, _ := signal["trail_percent"].(float64)
	timeInForce, _ := signal["time_in_force"].(string)

	if symbol == "" || side == "" || quantity <= 0 {
		o.logger.Warn().Interface("signal", signal).Msg("Invalid strategy signal")
		return
	}

	// Default to market order if type not specified
	if orderType == "" {
		orderType = OrderTypeMarket
	}

	// Default time in force
	if timeInForce == "" {
		timeInForce = "GTC"
	}

	// Handle different order types
	switch orderType {
	case OrderTypeTrailing:
		trailMsg := PlaceTrailingStopMsg{
			Symbol:       symbol,
			Side:         side,
			Quantity:     quantity,
			TrailAmount:  trailAmount,
			TrailPercent: trailPercent,
			Reason:       reason,
		}
		o.onPlaceTrailingStop(ctx, trailMsg)

	case OrderTypeStopMarket, OrderTypeStopLimit:
		stopMsg := PlaceStopOrderMsg{
			Symbol:     symbol,
			Side:       side,
			Quantity:   quantity,
			StopPrice:  stopPrice,
			LimitPrice: price,
			Reason:     reason,
		}
		o.onPlaceStopOrder(ctx, stopMsg)

	default:
		// Regular market or limit order
		orderMsg := PlaceOrderMsg{
			Symbol:       symbol,
			Side:         side,
			Type:         orderType,
			Quantity:     quantity,
			Price:        price,
			StopPrice:    stopPrice,
			TrailAmount:  trailAmount,
			TrailPercent: trailPercent,
			TimeInForce:  timeInForce,
			Reason:       reason,
		}
		o.onPlaceOrder(ctx, orderMsg)
	}
}

func (o *OrderManagerActor) onPlaceOrder(ctx *actor.Context, msg PlaceOrderMsg) {
	o.logger.Info().
		Str("symbol", msg.Symbol).
		Str("side", msg.Side).
		Str("type", msg.Type).
		Float64("quantity", msg.Quantity).
		Float64("price", msg.Price).
		Str("reason", msg.Reason).
		Msg("Placing order")

	if o.exchange == nil {
		o.logger.Error().Msg("No exchange interface available")
		ctx.Respond(fmt.Errorf("no exchange interface"))
		return
	}

	// Handle advanced order types
	switch msg.Type {
	case OrderTypeTrailing:
		o.onPlaceTrailingStop(ctx, PlaceTrailingStopMsg{
			Symbol:       msg.Symbol,
			Side:         msg.Side,
			Quantity:     msg.Quantity,
			TrailAmount:  msg.TrailAmount,
			TrailPercent: msg.TrailPercent,
			Reason:       msg.Reason,
		})
		return
	case OrderTypeStopMarket, OrderTypeStopLimit:
		o.onPlaceStopOrder(ctx, PlaceStopOrderMsg{
			Symbol:     msg.Symbol,
			Side:       msg.Side,
			Quantity:   msg.Quantity,
			StopPrice:  msg.StopPrice,
			LimitPrice: msg.Price, // For stop-limit orders
			Reason:     msg.Reason,
		})
		return
	}

	// Create enhanced order object
	enhancedOrder := &EnhancedOrder{
		Order: &exchanges.Order{
			Symbol:   msg.Symbol,
			Side:     msg.Side,
			Type:     msg.Type,
			Quantity: msg.Quantity,
			Price:    msg.Price,
			Status:   StatusPending,
			Time:     time.Now(),
		},
		OriginalType: msg.Type,
		TimeInForce:  msg.TimeInForce,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Place order through exchange
	orderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	placedOrder, err := o.exchange.PlaceOrder(orderCtx, enhancedOrder.Order)
	if err != nil {
		o.logger.Error().Err(err).Msg("Failed to place order")
		ctx.Respond(err)
		return
	}

	// Update enhanced order with exchange response
	enhancedOrder.Order = placedOrder
	enhancedOrder.UpdatedAt = time.Now()

	// Store order
	o.mutex.Lock()
	o.orders[placedOrder.ID] = enhancedOrder
	o.mutex.Unlock()

	// Persist order to database
	o.persistEnhancedOrder(enhancedOrder)

	o.logger.Info().
		Str("order_id", placedOrder.ID).
		Str("status", placedOrder.Status).
		Msg("Order placed successfully")

	ctx.Respond(enhancedOrder)
}

func (o *OrderManagerActor) onCancelOrder(ctx *actor.Context, msg CancelOrderMsg) {
	o.logger.Info().
		Str("order_id", msg.OrderID).
		Str("symbol", msg.Symbol).
		Msg("Canceling order")

	if o.exchange == nil {
		o.logger.Error().Msg("No exchange interface available")
		ctx.Respond(fmt.Errorf("no exchange interface"))
		return
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Check if it's a regular order
	if order, exists := o.orders[msg.OrderID]; exists {
		// Cancel order through exchange for regular orders
		if order.OriginalType == OrderTypeMarket || order.OriginalType == OrderTypeLimit {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := o.exchange.CancelOrder(cancelCtx, msg.Symbol, msg.OrderID)
			if err != nil {
				o.logger.Error().Err(err).Msg("Failed to cancel order")
				ctx.Respond(err)
				return
			}
		}

		// Update local order status
		order.Status = StatusCancelled
		order.UpdatedAt = time.Now()
		o.persistEnhancedOrder(order)

		o.logger.Info().Str("order_id", msg.OrderID).Msg("Order cancelled successfully")
		ctx.Respond("cancelled")
		return
	}

	// Check if it's a stop order
	if stopOrder, exists := o.stopOrders[msg.OrderID]; exists {
		stopOrder.Status = StatusCancelled
		stopOrder.UpdatedAt = time.Now()
		delete(o.stopOrders, msg.OrderID)
		o.persistEnhancedOrder(stopOrder)

		o.logger.Info().Str("order_id", msg.OrderID).Msg("Stop order cancelled successfully")
		ctx.Respond("cancelled")
		return
	}

	// Check if it's a trailing stop order
	if trailOrder, exists := o.trailingStops[msg.OrderID]; exists {
		trailOrder.Status = StatusCancelled
		trailOrder.UpdatedAt = time.Now()
		delete(o.trailingStops, msg.OrderID)
		o.persistEnhancedOrder(trailOrder)

		o.logger.Info().Str("order_id", msg.OrderID).Msg("Trailing stop order cancelled successfully")
		ctx.Respond("cancelled")
		return
	}

	// Order not found
	ctx.Respond(fmt.Errorf("order not found: %s", msg.OrderID))
}

func (o *OrderManagerActor) onGetOrders(ctx *actor.Context, msg GetOrdersMsg) {
	orders := make([]*EnhancedOrder, 0)

	o.mutex.RLock()
	for _, order := range o.orders {
		if msg.Symbol == "" || order.Symbol == msg.Symbol {
			orders = append(orders, order)
		}
	}
	o.mutex.RUnlock()

	ctx.Respond(orders)
}

func (o *OrderManagerActor) onOrderUpdate(ctx *actor.Context, msg OrderUpdateMsg) {
	// Update order status from exchange
	o.mutex.Lock()
	o.orders[msg.Order.ID] = msg.Order
	o.mutex.Unlock()

	o.persistEnhancedOrder(msg.Order)

	o.logger.Info().
		Str("order_id", msg.Order.ID).
		Str("status", msg.Order.Status).
		Msg("Order status updated")
}

func (o *OrderManagerActor) persistEnhancedOrder(order *EnhancedOrder) {
	// TODO: Implement database persistence for enhanced orders
	// Reduced chattiness - only log on errors or important state changes
	if order.Status == StatusFilled || order.Status == StatusCancelled {
		o.logger.Info().
			Str("order_id", order.ID).
			Str("status", order.Status).
			Str("type", order.OriginalType).
			Msg("Order status updated")
	}
}

func (o *OrderManagerActor) onStatus(ctx *actor.Context) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	activeOrders := 0
	for _, order := range o.orders {
		if order.Status == StatusPending || order.Status == StatusOpen || order.Status == StatusPartiallyFilled {
			activeOrders++
		}
	}

	pendingStopOrders := len(o.stopOrders)
	pendingTrailingStops := len(o.trailingStops)

	status := map[string]interface{}{
		"exchange":               o.exchangeName,
		"total_orders":           len(o.orders),
		"active_orders":          activeOrders,
		"pending_stop_orders":    pendingStopOrders,
		"pending_trailing_stops": pendingTrailingStops,
		"symbols_tracked":        len(o.priceCache),
		"timestamp":              time.Now(),
	}

	ctx.Respond(status)
}

func (o *OrderManagerActor) persistOrder(order *exchanges.Order) {
	// TODO: Implement database persistence
	// Reduced chattiness - only log important status changes
	if order.Status == "filled" || order.Status == "cancelled" {
		o.logger.Info().
			Str("order_id", order.ID).
			Str("status", order.Status).
			Msg("Order status persisted")
	}
}

// Advanced order management methods

func (o *OrderManagerActor) onPlaceTrailingStop(ctx *actor.Context, msg PlaceTrailingStopMsg) {
	o.logger.Info().
		Str("symbol", msg.Symbol).
		Str("side", msg.Side).
		Float64("quantity", msg.Quantity).
		Float64("trail_amount", msg.TrailAmount).
		Float64("trail_percent", msg.TrailPercent).
		Msg("Placing trailing stop order")

	// Get current market price
	currentPrice, exists := o.priceCache[msg.Symbol]
	if !exists {
		o.logger.Error().Str("symbol", msg.Symbol).Msg("No current price available for trailing stop")
		ctx.Respond(fmt.Errorf("no current price available for %s", msg.Symbol))
		return
	}

	// Create trailing stop order
	enhancedOrder := &EnhancedOrder{
		Order: &exchanges.Order{
			Symbol:   msg.Symbol,
			Side:     msg.Side,
			Type:     OrderTypeTrailing,
			Quantity: msg.Quantity,
			Status:   StatusPending,
			Time:     time.Now(),
		},
		OriginalType:  OrderTypeTrailing,
		TrailAmount:   msg.TrailAmount,
		TrailPercent:  msg.TrailPercent,
		HighWaterMark: currentPrice,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Generate unique order ID
	enhancedOrder.ID = fmt.Sprintf("trail_%d", time.Now().UnixNano())

	// Store in trailing stops
	o.mutex.Lock()
	o.trailingStops[enhancedOrder.ID] = enhancedOrder
	o.mutex.Unlock()

	o.logger.Info().
		Str("order_id", enhancedOrder.ID).
		Float64("initial_price", currentPrice).
		Msg("Trailing stop order created")

	ctx.Respond(enhancedOrder)
}

func (o *OrderManagerActor) onPlaceStopOrder(ctx *actor.Context, msg PlaceStopOrderMsg) {
	o.logger.Info().
		Str("symbol", msg.Symbol).
		Str("side", msg.Side).
		Float64("quantity", msg.Quantity).
		Float64("stop_price", msg.StopPrice).
		Float64("limit_price", msg.LimitPrice).
		Msg("Placing stop order")

	orderType := OrderTypeStopMarket
	if msg.LimitPrice > 0 {
		orderType = OrderTypeStopLimit
	}

	// Create stop order
	enhancedOrder := &EnhancedOrder{
		Order: &exchanges.Order{
			Symbol:   msg.Symbol,
			Side:     msg.Side,
			Type:     orderType,
			Quantity: msg.Quantity,
			Price:    msg.LimitPrice,
			Status:   StatusPending,
			Time:     time.Now(),
		},
		OriginalType: orderType,
		StopPrice:    msg.StopPrice,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Generate unique order ID
	enhancedOrder.ID = fmt.Sprintf("stop_%d", time.Now().UnixNano())

	// Store in stop orders
	o.mutex.Lock()
	o.stopOrders[enhancedOrder.ID] = enhancedOrder
	o.mutex.Unlock()

	o.logger.Info().
		Str("order_id", enhancedOrder.ID).
		Float64("stop_price", msg.StopPrice).
		Msg("Stop order created")

	ctx.Respond(enhancedOrder)
}

func (o *OrderManagerActor) onModifyOrder(ctx *actor.Context, msg ModifyOrderMsg) {
	o.logger.Info().
		Str("order_id", msg.OrderID).
		Str("symbol", msg.Symbol).
		Msg("Modifying order")

	o.mutex.Lock()
	order, exists := o.orders[msg.OrderID]
	if !exists {
		o.mutex.Unlock()
		ctx.Respond(fmt.Errorf("order not found: %s", msg.OrderID))
		return
	}

	// Update order fields
	if msg.NewQuantity != nil {
		order.Quantity = *msg.NewQuantity
	}
	if msg.NewPrice != nil {
		order.Price = *msg.NewPrice
	}
	if msg.NewStopPrice != nil {
		order.StopPrice = *msg.NewStopPrice
	}

	order.UpdatedAt = time.Now()
	o.mutex.Unlock()

	// If this is a regular exchange order, modify it on the exchange
	if order.OriginalType == OrderTypeMarket || order.OriginalType == OrderTypeLimit {
		// TODO: Implement exchange order modification
		o.logger.Info().Str("order_id", msg.OrderID).Msg("Exchange order modification not yet implemented")
	}

	o.persistEnhancedOrder(order)
	ctx.Respond(order)
}

func (o *OrderManagerActor) onPriceUpdate(ctx *actor.Context, msg PriceUpdateMsg) {
	o.mutex.Lock()
	o.priceCache[msg.Symbol] = msg.Price
	o.mutex.Unlock()
}

func (o *OrderManagerActor) checkStopOrders(ctx *actor.Context) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	for orderID, stopOrder := range o.stopOrders {
		if stopOrder.IsTriggered {
			continue
		}

		currentPrice, exists := o.priceCache[stopOrder.Symbol]
		if !exists {
			continue
		}

		shouldTrigger := false

		// Check trigger conditions
		if stopOrder.Side == "buy" {
			// Buy stop: trigger when price rises above stop price
			shouldTrigger = currentPrice >= stopOrder.StopPrice
		} else {
			// Sell stop: trigger when price falls below stop price
			shouldTrigger = currentPrice <= stopOrder.StopPrice
		}

		if shouldTrigger {
			o.triggerStopOrder(ctx, orderID, stopOrder, currentPrice)
		}
	}
}

func (o *OrderManagerActor) updateTrailingStops(ctx *actor.Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for orderID, trailOrder := range o.trailingStops {
		currentPrice, exists := o.priceCache[trailOrder.Symbol]
		if !exists {
			continue
		}

		// Update high water mark
		if trailOrder.Side == "sell" {
			// For sell trailing stop, track highest price
			if currentPrice > trailOrder.HighWaterMark {
				trailOrder.HighWaterMark = currentPrice
				trailOrder.UpdatedAt = time.Now()
			}

			// Calculate trigger price
			var triggerPrice float64
			if trailOrder.TrailPercent > 0 {
				triggerPrice = trailOrder.HighWaterMark * (1 - trailOrder.TrailPercent/100)
			} else {
				triggerPrice = trailOrder.HighWaterMark - trailOrder.TrailAmount
			}

			// Check if we should trigger
			if currentPrice <= triggerPrice {
				o.triggerTrailingStop(ctx, orderID, trailOrder, currentPrice)
			}
		} else {
			// For buy trailing stop, track lowest price
			if currentPrice < trailOrder.HighWaterMark {
				trailOrder.HighWaterMark = currentPrice
				trailOrder.UpdatedAt = time.Now()
			}

			// Calculate trigger price
			var triggerPrice float64
			if trailOrder.TrailPercent > 0 {
				triggerPrice = trailOrder.HighWaterMark * (1 + trailOrder.TrailPercent/100)
			} else {
				triggerPrice = trailOrder.HighWaterMark + trailOrder.TrailAmount
			}

			// Check if we should trigger
			if currentPrice >= triggerPrice {
				o.triggerTrailingStop(ctx, orderID, trailOrder, currentPrice)
			}
		}
	}
}

func (o *OrderManagerActor) triggerStopOrder(ctx *actor.Context, orderID string, stopOrder *EnhancedOrder, currentPrice float64) {
	o.logger.Info().
		Str("order_id", orderID).
		Float64("trigger_price", currentPrice).
		Float64("stop_price", stopOrder.StopPrice).
		Msg("Triggering stop order")

	stopOrder.IsTriggered = true
	stopOrder.TriggerPrice = currentPrice
	stopOrder.UpdatedAt = time.Now()

	// Create market order to execute the stop
	marketOrder := &exchanges.Order{
		Symbol:   stopOrder.Symbol,
		Side:     stopOrder.Side,
		Type:     OrderTypeMarket,
		Quantity: stopOrder.Quantity,
		Status:   StatusPending,
		Time:     time.Now(),
	}

	if stopOrder.OriginalType == OrderTypeStopLimit && stopOrder.Price > 0 {
		marketOrder.Type = OrderTypeLimit
		marketOrder.Price = stopOrder.Price
	}

	// Place the market order
	orderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	placedOrder, err := o.exchange.PlaceOrder(orderCtx, marketOrder)
	if err != nil {
		o.logger.Error().Err(err).Str("order_id", orderID).Msg("Failed to execute stop order")
		return
	}

	// Update stop order status
	stopOrder.Status = StatusFilled
	stopOrder.Order.ID = placedOrder.ID

	// Move from stopOrders to orders
	delete(o.stopOrders, orderID)
	o.orders[placedOrder.ID] = stopOrder

	o.persistEnhancedOrder(stopOrder)
}

func (o *OrderManagerActor) triggerTrailingStop(ctx *actor.Context, orderID string, trailOrder *EnhancedOrder, currentPrice float64) {
	o.logger.Info().
		Str("order_id", orderID).
		Float64("trigger_price", currentPrice).
		Float64("high_water_mark", trailOrder.HighWaterMark).
		Msg("Triggering trailing stop order")

	trailOrder.IsTriggered = true
	trailOrder.TriggerPrice = currentPrice
	trailOrder.UpdatedAt = time.Now()

	// Create market order to execute the trailing stop
	marketOrder := &exchanges.Order{
		Symbol:   trailOrder.Symbol,
		Side:     trailOrder.Side,
		Type:     OrderTypeMarket,
		Quantity: trailOrder.Quantity,
		Status:   StatusPending,
		Time:     time.Now(),
	}

	// Place the market order
	orderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	placedOrder, err := o.exchange.PlaceOrder(orderCtx, marketOrder)
	if err != nil {
		o.logger.Error().Err(err).Str("order_id", orderID).Msg("Failed to execute trailing stop order")
		return
	}

	// Update trailing stop order status
	trailOrder.Status = StatusFilled
	trailOrder.Order.ID = placedOrder.ID

	// Move from trailingStops to orders
	delete(o.trailingStops, orderID)
	o.orders[placedOrder.ID] = trailOrder

	o.persistEnhancedOrder(trailOrder)
}
