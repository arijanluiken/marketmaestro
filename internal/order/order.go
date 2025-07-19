package order

import (
	"context"
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

// Messages for order manager actor communication
type (
	PlaceOrderMsg struct {
		Symbol   string
		Side     string // "buy" or "sell"
		Type     string // "market" or "limit"
		Quantity float64
		Price    float64
		Reason   string
	}
	CancelOrderMsg struct {
		OrderID string
		Symbol  string
	}
	GetOrdersMsg   struct{ Symbol string }
	OrderUpdateMsg struct{ Order *exchanges.Order }
	StatusMsg      struct{}
)

// OrderManagerActor manages order placement and advanced order types
type OrderManagerActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
	orders       map[string]*exchanges.Order // Active orders by ID
	exchange     exchanges.Exchange          // Reference to exchange interface
}

// New creates a new order manager actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *OrderManagerActor {
	return &OrderManagerActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
		orders:       make(map[string]*exchanges.Order),
	}
}

// SetExchange sets the exchange interface for order operations
func (o *OrderManagerActor) SetExchange(exchange exchanges.Exchange) {
	o.exchange = exchange
}

// Receive handles incoming messages
func (o *OrderManagerActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		o.onStarted(ctx)
	case actor.Stopped:
		o.onStopped(ctx)
	case PlaceOrderMsg:
		o.onPlaceOrder(ctx, msg)
	case CancelOrderMsg:
		o.onCancelOrder(ctx, msg)
	case GetOrdersMsg:
		o.onGetOrders(ctx, msg)
	case OrderUpdateMsg:
		o.onOrderUpdate(ctx, msg)
	case StatusMsg:
		o.onStatus(ctx)
	case map[string]interface{}: // Handle strategy signals
		o.onStrategySignal(ctx, msg)
	default:
		o.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (o *OrderManagerActor) onStarted(ctx *actor.Context) {
	o.logger.Info().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor started")
}

func (o *OrderManagerActor) onStopped(ctx *actor.Context) {
	o.logger.Info().
		Str("exchange", o.exchangeName).
		Msg("Order manager actor stopped")
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

	// Convert strategy signal to order message
	symbol, _ := signal["symbol"].(string)
	side, _ := signal["side"].(string)
	orderType, _ := signal["type"].(string)
	quantity, _ := signal["quantity"].(float64)
	price, _ := signal["price"].(float64)
	reason, _ := signal["reason"].(string)

	if symbol == "" || side == "" || quantity <= 0 {
		o.logger.Warn().Interface("signal", signal).Msg("Invalid strategy signal")
		return
	}

	// Default to market order if type not specified
	if orderType == "" {
		orderType = "market"
	}

	orderMsg := PlaceOrderMsg{
		Symbol:   symbol,
		Side:     side,
		Type:     orderType,
		Quantity: quantity,
		Price:    price,
		Reason:   reason,
	}

	o.onPlaceOrder(ctx, orderMsg)
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

	// Create order object
	order := &exchanges.Order{
		Symbol:   msg.Symbol,
		Side:     msg.Side,
		Type:     msg.Type,
		Quantity: msg.Quantity,
		Price:    msg.Price,
		Status:   "pending",
		Time:     time.Now(),
	}

	// Place order through exchange
	orderCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	placedOrder, err := o.exchange.PlaceOrder(orderCtx, order)
	if err != nil {
		o.logger.Error().Err(err).Msg("Failed to place order")
		ctx.Respond(err)
		return
	}

	// Store order
	o.orders[placedOrder.ID] = placedOrder

	// Persist order to database
	o.persistOrder(placedOrder)

	o.logger.Info().
		Str("order_id", placedOrder.ID).
		Str("status", placedOrder.Status).
		Msg("Order placed successfully")

	ctx.Respond(placedOrder)
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

	// Cancel order through exchange
	cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := o.exchange.CancelOrder(cancelCtx, msg.Symbol, msg.OrderID)
	if err != nil {
		o.logger.Error().Err(err).Msg("Failed to cancel order")
		ctx.Respond(err)
		return
	}

	// Update local order status
	if order, exists := o.orders[msg.OrderID]; exists {
		order.Status = "cancelled"
		o.persistOrder(order)
	}

	o.logger.Info().Str("order_id", msg.OrderID).Msg("Order cancelled successfully")
	ctx.Respond("cancelled")
}

func (o *OrderManagerActor) onGetOrders(ctx *actor.Context, msg GetOrdersMsg) {
	orders := make([]*exchanges.Order, 0)

	for _, order := range o.orders {
		if msg.Symbol == "" || order.Symbol == msg.Symbol {
			orders = append(orders, order)
		}
	}

	ctx.Respond(orders)
}

func (o *OrderManagerActor) onOrderUpdate(ctx *actor.Context, msg OrderUpdateMsg) {
	// Update order status from exchange
	o.orders[msg.Order.ID] = msg.Order
	o.persistOrder(msg.Order)

	o.logger.Info().
		Str("order_id", msg.Order.ID).
		Str("status", msg.Order.Status).
		Msg("Order status updated")
}

func (o *OrderManagerActor) onStatus(ctx *actor.Context) {
	activeOrders := 0
	for _, order := range o.orders {
		if order.Status == "pending" || order.Status == "open" || order.Status == "partially_filled" {
			activeOrders++
		}
	}

	status := map[string]interface{}{
		"exchange":      o.exchangeName,
		"total_orders":  len(o.orders),
		"active_orders": activeOrders,
		"timestamp":     time.Now(),
	}

	ctx.Respond(status)
}

func (o *OrderManagerActor) persistOrder(order *exchanges.Order) {
	// TODO: Implement database persistence
	o.logger.Debug().
		Str("order_id", order.ID).
		Str("status", order.Status).
		Msg("Order persisted to database")
}
