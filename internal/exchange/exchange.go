package exchange

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/internal/order"
	"github.com/arijanluiken/mercantile/internal/portfolio"
	"github.com/arijanluiken/mercantile/internal/risk"
	"github.com/arijanluiken/mercantile/internal/settings"
	"github.com/arijanluiken/mercantile/internal/strategy"
	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

// Messages for exchange actor communication
type (
	ConnectMessage     struct{}
	DisconnectMessage  struct{}
	SubscribeKlinesMsg struct {
		Symbols  []string
		Interval string
	}
	SubscribeOrderBookMsg struct{ Symbols []string }
	GetBalancesMsg        struct{}
	GetPositionsMsg       struct{}
	GetStrategiesMsg      struct{}
	StatusMsg             struct{}

	// Notification messages
	PortfolioActorCreatedMsg struct {
		Exchange     string
		PortfolioPID *actor.PID
	}
	SetAPIActorMsg struct {
		APIActorPID *actor.PID
	}

	// Strategy data message that can be sent to API
	StrategyDataUpdateMsg struct {
		Exchange   string
		Strategies []map[string]interface{}
	}

	// Portfolio data message that can be sent to API
	PortfolioDataUpdateMsg struct {
		Exchange  string
		Balances  []map[string]interface{}
		Positions []map[string]interface{}
	}

	// Orders data message that can be sent to API
	OrdersDataUpdateMsg struct {
		Exchange string
		Orders   []map[string]interface{}
	}

	// Data messages
	KlineDataMsg      struct{ Kline *exchanges.Kline }
	OrderBookDataMsg  struct{ OrderBook *exchanges.OrderBook }
	OrderUpdateMsg    struct{ Order *exchanges.Order }
	PositionUpdateMsg struct{ Position *exchanges.Position }
	BalanceUpdateMsg  struct{ Balance *exchanges.Balance }
)

// ExchangeActor manages exchange connections and child actors
type ExchangeActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
	exchange     exchanges.Exchange
	factory      *exchanges.Factory

	// Child actors
	strategyActors  map[string]*actor.PID
	orderManagerPID *actor.PID
	riskManagerPID  *actor.PID
	portfolioPID    *actor.PID
	settingsPID     *actor.PID
	apiActorPID     *actor.PID

	// State
	connected            bool
	subscribedKlines     map[string]bool
	subscribedOrderBooks map[string]bool

	// Store actor system for sending messages from callbacks
	actorSystem *actor.Engine
}

// New creates a new exchange actor
func New(exchangeName string, exchangeConfig map[string]interface{}, cfg *config.Config, db *database.DB, logger zerolog.Logger) *ExchangeActor {
	factory := exchanges.NewFactory(logger)

	return &ExchangeActor{
		exchangeName:         exchangeName,
		config:               cfg,
		db:                   db,
		logger:               logger,
		factory:              factory,
		strategyActors:       make(map[string]*actor.PID),
		subscribedKlines:     make(map[string]bool),
		subscribedOrderBooks: make(map[string]bool),
	}
}

// Receive handles incoming messages
func (e *ExchangeActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		e.onStarted(ctx)
	case actor.Stopped:
		e.onStopped(ctx)
	case actor.Initialized:
		e.onInitialized(ctx)
	case ConnectMessage:
		e.onConnect(ctx)
	case DisconnectMessage:
		e.onDisconnect(ctx)
	case SubscribeKlinesMsg:
		e.onSubscribeKlines(ctx, msg)
	case SubscribeOrderBookMsg:
		e.onSubscribeOrderBook(ctx, msg)
	case GetBalancesMsg:
		e.onGetBalances(ctx)
	case GetPositionsMsg:
		e.onGetPositions(ctx)
	case GetStrategiesMsg:
		e.onGetStrategies(ctx)
	case StatusMsg:
		e.onStatus(ctx)
	case KlineDataMsg:
		e.onKlineData(ctx, msg)
	case OrderBookDataMsg:
		e.onOrderBookData(ctx, msg)
	// Portfolio actor messages
	case portfolio.GetBalancesMsg:
		e.onPortfolioRequestBalances(ctx)
	case portfolio.GetPositionsMsg:
		e.onPortfolioRequestPositions(ctx)
	case SetAPIActorMsg:
		e.onSetAPIActor(ctx, msg)
	default:
		e.logger.Warn().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received unknown message")
	}
}

func (e *ExchangeActor) onStarted(ctx *actor.Context) {
	e.logger.Info().Str("exchange", e.exchangeName).Msg("Exchange actor started")

	// Store actor system for sending messages from callbacks
	e.actorSystem = ctx.Engine()

	// Start child actors
	e.startChildActors(ctx)

	// Auto-connect to exchange
	ctx.Send(ctx.PID(), ConnectMessage{})

	// Start configured strategies
	e.startConfiguredStrategies(ctx)
}

func (e *ExchangeActor) onInitialized(ctx *actor.Context) {
	e.logger.Debug().Str("exchange", e.exchangeName).Msg("Exchange actor initialized")
}

func (e *ExchangeActor) startConfiguredStrategies(ctx *actor.Context) {
	// Check if this exchange is configured
	exchangeConfig, exists := e.config.Exchanges[e.exchangeName]
	if !exists || !exchangeConfig.Enabled {
		e.logger.Info().Str("exchange", e.exchangeName).Msg("Exchange not enabled in configuration")
		return
	}

	// Create strategy engine to extract intervals from scripts
	strategyEngine := strategy.NewStrategyEngine(e.logger)

	// Start strategies for each configured pair
	for _, pairConfig := range exchangeConfig.Pairs {
		symbols := []string{pairConfig.Symbol}

		// Collect unique intervals from all strategies for this symbol
		intervals := make(map[string]bool)
		for _, strategyConfig := range pairConfig.Strategies {
			// Get interval from strategy script
			interval, err := strategyEngine.GetStrategyInterval(strategyConfig.Name)
			if err != nil {
				e.logger.Error().
					Err(err).
					Str("strategy", strategyConfig.Name).
					Msg("Failed to get strategy interval, using default")
				interval = "1m" // Default fallback
			}
			intervals[interval] = true
		}

		// Subscribe to klines for each unique interval
		for interval := range intervals {
			ctx.Send(ctx.PID(), SubscribeKlinesMsg{
				Symbols:  symbols,
				Interval: interval,
			})
		}

		// Subscribe to order book for this symbol (only once)
		ctx.Send(ctx.PID(), SubscribeOrderBookMsg{
			Symbols: symbols,
		})

		// Start each strategy for this pair
		for _, strategyConfig := range pairConfig.Strategies {
			err := e.StartStrategy(ctx, strategyConfig.Name, pairConfig.Symbol, strategyConfig.Config)
			if err != nil {
				e.logger.Error().
					Err(err).
					Str("strategy", strategyConfig.Name).
					Str("symbol", pairConfig.Symbol).
					Msg("Failed to start strategy")
			}
		}
	}
}

func (e *ExchangeActor) onStopped(ctx *actor.Context) {
	e.logger.Info().Str("exchange", e.exchangeName).Msg("Exchange actor stopped")

	if e.exchange != nil && e.connected {
		e.exchange.Disconnect()
	}
}

func (e *ExchangeActor) startChildActors(ctx *actor.Context) {
	// Start Order Manager Actor
	orderManagerPID := ctx.SpawnChild(func() actor.Receiver {
		orderManager := order.New(e.exchangeName, e.config, e.db, e.logger.With().Str("actor", "order_manager").Logger())
		return orderManager
	}, "order_manager")
	e.orderManagerPID = orderManagerPID

	// Start Risk Manager Actor
	riskManagerPID := ctx.SpawnChild(func() actor.Receiver {
		return risk.New(e.exchangeName, e.config, e.db, e.logger.With().Str("actor", "risk_manager").Logger())
	}, "risk_manager")
	e.riskManagerPID = riskManagerPID

	// Start Portfolio Actor
	portfolioPID := ctx.SpawnChild(func() actor.Receiver {
		return portfolio.New(e.exchangeName, e.config, e.db, e.logger.With().Str("actor", "portfolio").Logger())
	}, "portfolio")
	e.portfolioPID = portfolioPID

	// Send exchange actor reference to portfolio actor
	ctx.Send(portfolioPID, portfolio.SetExchangeActorMsg{ExchangeActorPID: ctx.PID()})

	// Notify supervisor about portfolio actor creation
	if ctx.Parent() != nil {
		ctx.Send(ctx.Parent(), PortfolioActorCreatedMsg{
			Exchange:     e.exchangeName,
			PortfolioPID: portfolioPID,
		})
	}

	// Start Settings Actor
	settingsPID := ctx.SpawnChild(func() actor.Receiver {
		return settings.New(e.exchangeName, e.config, e.db, e.logger.With().Str("actor", "settings").Logger())
	}, "settings")
	e.settingsPID = settingsPID

	e.logger.Info().Msg("Child actors started successfully")
}

func (e *ExchangeActor) onConnect(ctx *actor.Context) {
	if e.connected {
		e.logger.Warn().Msg("Already connected to exchange")
		return
	}

	// Create exchange instance
	exchangeConfig := map[string]interface{}{
		"api_key": "", // These would come from config
		"secret":  "",
		"testnet": false,
	}

	if e.exchangeName == "bybit" {
		exchangeConfig["api_key"] = e.config.BybitAPIKey
		exchangeConfig["secret"] = e.config.BybitSecret
		exchangeConfig["testnet"] = e.config.BybitTestnet
	} else if e.exchangeName == "bitvavo" {
		exchangeConfig["api_key"] = e.config.BitvavoAPIKey
		exchangeConfig["secret"] = e.config.BitvavoSecret
		exchangeConfig["testnet"] = e.config.BitvavoTestnet
	}

	exchange, err := e.factory.CreateExchange(e.exchangeName, exchangeConfig)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to create exchange instance")
		return
	}

	e.exchange = exchange

	// Connect to exchange
	connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.exchange.Connect(connectCtx); err != nil {
		e.logger.Error().Err(err).Msg("Failed to connect to exchange")
		return
	}

	e.connected = true
	e.logger.Info().Msg("Successfully connected to exchange")

	// Provide exchange interface to order manager
	if e.orderManagerPID != nil {
		orderManagerSetExchangeMsg := map[string]interface{}{
			"action":   "set_exchange",
			"exchange": e.exchange,
		}
		ctx.Send(e.orderManagerPID, orderManagerSetExchangeMsg)
	}
}

func (e *ExchangeActor) onDisconnect(ctx *actor.Context) {
	if !e.connected {
		e.logger.Warn().Msg("Not connected to exchange")
		return
	}

	if err := e.exchange.Disconnect(); err != nil {
		e.logger.Error().Err(err).Msg("Error disconnecting from exchange")
	}

	e.connected = false
	e.logger.Info().Msg("Disconnected from exchange")
}

func (e *ExchangeActor) onSubscribeKlines(ctx *actor.Context, msg SubscribeKlinesMsg) {
	if !e.connected {
		e.logger.Error().Msg("Cannot subscribe to klines: not connected")
		return
	}

	e.logger.Info().
		Strs("symbols", msg.Symbols).
		Str("interval", msg.Interval).
		Msg("Subscribing to klines")

	// Subscribe to klines via exchange WebSocket with this actor as handler
	err := e.exchange.SubscribeKlines(context.Background(), msg.Symbols, msg.Interval, e)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to subscribe to klines")
		return
	}

	// Track subscriptions
	for _, symbol := range msg.Symbols {
		e.subscribedKlines[symbol+":"+msg.Interval] = true
	}
}

func (e *ExchangeActor) onSubscribeOrderBook(ctx *actor.Context, msg SubscribeOrderBookMsg) {
	if !e.connected {
		e.logger.Error().Msg("Cannot subscribe to order book: not connected")
		return
	}

	e.logger.Info().
		Strs("symbols", msg.Symbols).
		Msg("Subscribing to order book")

	// Subscribe to order book via exchange WebSocket with this actor as handler
	err := e.exchange.SubscribeOrderBook(context.Background(), msg.Symbols, e)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to subscribe to order book")
		return
	}

	for _, symbol := range msg.Symbols {
		e.subscribedOrderBooks[symbol] = true
	}
}

// DataHandler interface implementation
func (e *ExchangeActor) OnKline(kline *exchanges.Kline) {
	// Info level only for important price updates
	e.logger.Info().
		Str("symbol", kline.Symbol).
		Str("interval", kline.Interval).
		Float64("close", kline.Close).
		Time("timestamp", kline.Timestamp).
		Msg("Received kline data")

	// Broadcast to all strategy actors using the actor system
	for _, strategyPID := range e.strategyActors {
		if strategyPID != nil && e.actorSystem != nil {
			e.actorSystem.Send(strategyPID, strategy.KlineDataMsg{Kline: kline})
		}
	}

	// Update portfolio with current market prices
	if e.portfolioPID != nil && e.actorSystem != nil {
		priceUpdate := portfolio.UpdateMarketPricesMsg{
			Prices: map[string]float64{
				kline.Symbol: kline.Close,
			},
		}
		e.actorSystem.Send(e.portfolioPID, priceUpdate)
	}

	// Send price update to order manager for stop/trailing orders
	if e.orderManagerPID != nil && e.actorSystem != nil {
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": kline.Symbol,
			"price":  kline.Close,
		}
		e.actorSystem.Send(e.orderManagerPID, priceUpdate)
	}
}

func (e *ExchangeActor) OnOrderBook(orderBook *exchanges.OrderBook) {
	// Reduced logging to only error cases
	if len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		e.logger.Warn().
			Str("symbol", orderBook.Symbol).
			Int("bids", len(orderBook.Bids)).
			Int("asks", len(orderBook.Asks)).
			Msg("Empty order book received")
	}

	// Broadcast to strategy actors using the actor system
	for _, strategyPID := range e.strategyActors {
		if strategyPID != nil && e.actorSystem != nil {
			e.actorSystem.Send(strategyPID, strategy.OrderBookDataMsg{OrderBook: orderBook})
		}
	}

	// Send price update to order manager for stop/trailing orders
	if e.orderManagerPID != nil && e.actorSystem != nil && len(orderBook.Bids) > 0 && len(orderBook.Asks) > 0 {
		// Use mid price for order management
		midPrice := (orderBook.Bids[0].Price + orderBook.Asks[0].Price) / 2
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": orderBook.Symbol,
			"price":  midPrice,
		}
		e.actorSystem.Send(e.orderManagerPID, priceUpdate)
	}
}

func (e *ExchangeActor) OnTicker(ticker *exchanges.Ticker) {
	// Only log significant price changes or errors

	// Broadcast to strategy actors using the actor system
	msg := strategy.TickerDataMsg{Ticker: ticker}
	for _, strategyPID := range e.strategyActors {
		if strategyPID != nil && e.actorSystem != nil {
			e.actorSystem.Send(strategyPID, msg)
		}
	}

	// Send price update to order manager for stop/trailing orders
	if e.orderManagerPID != nil && e.actorSystem != nil {
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": ticker.Symbol,
			"price":  ticker.Price,
		}
		e.actorSystem.Send(e.orderManagerPID, priceUpdate)
	}
}

func (e *ExchangeActor) onGetBalances(ctx *actor.Context) {
	e.logger.Debug().Bool("connected", e.connected).Msg("GetBalances request received")
	
	if !e.connected {
		e.logger.Error().Msg("Cannot get balances: not connected")
		ctx.Respond(fmt.Errorf("not connected"))
		return
	}

	e.logger.Info().Msg("Fetching balances from exchange")
	balanceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balances, err := e.exchange.GetBalances(balanceCtx)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to get balances")
		ctx.Respond(err)
		return
	}

	e.logger.Info().Int("balance_count", len(balances)).Msg("Successfully retrieved balances")

	// Send response to requester
	ctx.Respond(balances)

	// Also notify portfolio actor if balances are requested
	if e.portfolioPID != nil {
		for _, balance := range balances {
			portfolioMsg := portfolio.UpdateBalanceMsg{
				Exchange: e.exchangeName,
				Asset:    balance.Asset,
				Amount:   balance.Available,
			}
			ctx.Send(e.portfolioPID, portfolioMsg)
		}
	}

	// Convert balances to map format for API
	balanceData := make([]map[string]interface{}, len(balances))
	for i, balance := range balances {
		balanceData[i] = map[string]interface{}{
			"asset":     balance.Asset,
			"total":     balance.Total,
			"available": balance.Available,
			"locked":    balance.Locked,
		}
	}

	// Send balance data to API actor if available
	if e.apiActorPID != nil {
		// Get current positions to send combined portfolio data
		e.sendPortfolioDataToAPI(ctx, balanceData, nil)
	}
}

func (e *ExchangeActor) onGetPositions(ctx *actor.Context) {
	if !e.connected {
		e.logger.Error().Msg("Cannot get positions: not connected")
		ctx.Respond(fmt.Errorf("not connected"))
		return
	}

	positionCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	positions, err := e.exchange.GetPositions(positionCtx)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to get positions")
		ctx.Respond(err)
		return
	}

	// Send response to requester
	ctx.Respond(positions)

	// Also notify portfolio actor if positions are requested
	if e.portfolioPID != nil {
		for _, position := range positions {
			portfolioMsg := portfolio.UpdatePositionMsg{
				Exchange: e.exchangeName,
				Symbol:   position.Symbol,
				Quantity: position.Size,
				Price:    position.EntryPrice,
				Side:     position.Side,
			}
			ctx.Send(e.portfolioPID, portfolioMsg)
		}
	}

	// Convert positions to map format for API
	positionData := make([]map[string]interface{}, len(positions))
	for i, position := range positions {
		positionData[i] = map[string]interface{}{
			"symbol":         position.Symbol,
			"size":           position.Size,
			"side":           position.Side,
			"entry_price":    position.EntryPrice,
			"mark_price":     position.MarkPrice,
			"unrealized_pnl": position.UnrealizedPL,
		}
	}

	// Send position data to API actor if available
	if e.apiActorPID != nil {
		e.sendPortfolioDataToAPI(ctx, nil, positionData)
	}
}

func (e *ExchangeActor) onGetStrategies(ctx *actor.Context) {
	strategies := make([]map[string]interface{}, 0)

	// Collect strategy information from all strategy actors
	for strategyKey, strategyPID := range e.strategyActors {
		// Parse strategy key to extract strategy name and symbol
		// Format: "strategyName:symbol"
		parts := strings.Split(strategyKey, ":")
		if len(parts) != 2 {
			continue
		}

		strategyName := parts[0]
		symbol := parts[1]

		// Create strategy info - for now we assume all tracked strategies are running
		// TODO: Add proper status tracking and PnL calculation
		strategyInfo := map[string]interface{}{
			"id":       fmt.Sprintf("%s:%s:%s", e.exchangeName, symbol, strategyName),
			"name":     strategyName,
			"symbol":   symbol,
			"exchange": e.exchangeName,
			"status":   "running", // Since we only track active strategy PIDs
			"pnl":      "$0.00",   // TODO: Calculate actual PnL from trades/positions
		}

		// Check if PID is still valid (actor still running)
		if strategyPID != nil {
			strategies = append(strategies, strategyInfo)
		}
	}

	// Respond to the caller (could be API or other actor)
	ctx.Respond(map[string]interface{}{"strategies": strategies})

	// Also send update to API actor if available
	if e.apiActorPID != nil {
		ctx.Send(e.apiActorPID, StrategyDataUpdateMsg{
			Exchange:   e.exchangeName,
			Strategies: strategies,
		})
		e.logger.Debug().
			Str("exchange", e.exchangeName).
			Int("strategy_count", len(strategies)).
			Msg("Sent strategy data to API actor")
	}
}

func (e *ExchangeActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":              e.exchangeName,
		"connected":             e.connected,
		"subscribed_klines":     len(e.subscribedKlines),
		"subscribed_orderbooks": len(e.subscribedOrderBooks),
		"strategy_actors":       len(e.strategyActors),
		"timestamp":             time.Now(),
	}

	ctx.Respond(status)
}

func (e *ExchangeActor) onKlineData(ctx *actor.Context, msg KlineDataMsg) {
	// Forward kline data to strategy actors for the same symbol
	for strategyKey, strategyPID := range e.strategyActors {
		// Extract symbol from strategy key (format: "strategy:symbol")
		if len(strategyKey) > 0 && fmt.Sprintf(":%s", msg.Kline.Symbol) == strategyKey[len(strategyKey)-len(msg.Kline.Symbol)-1:] {
			ctx.Send(strategyPID, strategy.KlineDataMsg{Kline: msg.Kline})
		}
	}
}

func (e *ExchangeActor) onOrderBookData(ctx *actor.Context, msg OrderBookDataMsg) {
	// Forward order book data to strategy actors for the same symbol
	for strategyKey, strategyPID := range e.strategyActors {
		// Extract symbol from strategy key (format: "strategy:symbol")
		if len(strategyKey) > 0 && fmt.Sprintf(":%s", msg.OrderBook.Symbol) == strategyKey[len(strategyKey)-len(msg.OrderBook.Symbol)-1:] {
			ctx.Send(strategyPID, strategy.OrderBookDataMsg{OrderBook: msg.OrderBook})
		}
	}
}

// StartStrategy starts a new strategy actor for a symbol
func (e *ExchangeActor) StartStrategy(ctx *actor.Context, strategyName, symbol string, config map[string]interface{}) error {
	strategyKey := fmt.Sprintf("%s:%s", strategyName, symbol)

	if _, exists := e.strategyActors[strategyKey]; exists {
		return fmt.Errorf("strategy %s already running for symbol %s", strategyName, symbol)
	}

	strategyPID := ctx.SpawnChild(func() actor.Receiver {
		strategyActor := strategy.New(
			strategyName,
			symbol,
			e.exchangeName,
			config,
			e.config,
			e.db,
			e.logger.With().Str("actor", "strategy").Str("strategy", strategyName).Str("symbol", symbol).Logger(),
		)
		// Set parent actor references for communication
		strategyActor.SetParentActors(e.orderManagerPID, e.riskManagerPID)
		return strategyActor
	}, strategyKey)

	e.strategyActors[strategyKey] = strategyPID
	e.logger.Info().
		Str("strategy", strategyName).
		Str("symbol", symbol).
		Msg("Strategy actor started")

	return nil
}

// NotifyTradeExecution notifies the portfolio actor when a trade is executed
func (e *ExchangeActor) NotifyTradeExecution(order *exchanges.Order) {
	if e.portfolioPID == nil {
		return
	}

	// Convert order to trade for portfolio tracking
	trade := portfolio.Trade{
		ID:        order.ID,
		Exchange:  e.exchangeName,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  order.Quantity,
		Price:     order.Price,
		Fee:       0.0, // Could be calculated based on exchange fees
		Timestamp: order.Time,
	}

	msg := portfolio.TradeExecutedMsg{Trade: trade}
	if e.actorSystem != nil {
		e.actorSystem.Send(e.portfolioPID, msg)
	}

	e.logger.Info().
		Str("order_id", order.ID).
		Str("symbol", order.Symbol).
		Str("side", order.Side).
		Float64("quantity", order.Quantity).
		Float64("price", order.Price).
		Msg("Trade execution notified to portfolio")
}

// Portfolio-specific request handlers
func (e *ExchangeActor) onPortfolioRequestBalances(ctx *actor.Context) {
	if !e.connected {
		e.logger.Error().Msg("Cannot get balances for portfolio: not connected")
		return
	}

	// Removed debug logging for portfolio requests

	balanceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balances, err := e.exchange.GetBalances(balanceCtx)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to get balances for portfolio")
		return
	}

	// Send balance updates to portfolio actor
	if e.portfolioPID != nil {
		for _, balance := range balances {
			portfolioMsg := portfolio.UpdateBalanceMsg{
				Exchange: e.exchangeName,
				Asset:    balance.Asset,
				Amount:   balance.Available,
			}
			ctx.Send(e.portfolioPID, portfolioMsg)
		}
		// Reduced logging - only log on errors
		if len(balances) == 0 {
			e.logger.Warn().Msg("No balances received from exchange")
		}
	}
}

func (e *ExchangeActor) onPortfolioRequestPositions(ctx *actor.Context) {
	if !e.connected {
		e.logger.Error().Msg("Cannot get positions for portfolio: not connected")
		return
	}

	// Removed debug logging for position requests

	positionCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	positions, err := e.exchange.GetPositions(positionCtx)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to get positions for portfolio")
		return
	}

	// Send position updates to portfolio actor
	if e.portfolioPID != nil {
		for _, position := range positions {
			portfolioMsg := portfolio.UpdatePositionMsg{
				Exchange: e.exchangeName,
				Symbol:   position.Symbol,
				Quantity: position.Size,
				Price:    position.EntryPrice,
				Side:     position.Side,
			}
			ctx.Send(e.portfolioPID, portfolioMsg)
		}
		// Only log if no positions received (potential issue)
		if len(positions) == 0 {
			e.logger.Info().Msg("No active positions found")
		}
	}
}

func (e *ExchangeActor) onSetAPIActor(ctx *actor.Context, msg SetAPIActorMsg) {
	e.apiActorPID = msg.APIActorPID
	e.logger.Info().Msg("API actor reference set")
}

// sendPortfolioDataToAPI sends portfolio data (balances and positions) to the API actor
func (e *ExchangeActor) sendPortfolioDataToAPI(ctx *actor.Context, balances []map[string]interface{}, positions []map[string]interface{}) {
	if e.apiActorPID == nil {
		return
	}

	// If we only have balances or positions, send what we have
	// The API will merge data as needed
	portfolioMsg := PortfolioDataUpdateMsg{
		Exchange:  e.exchangeName,
		Balances:  balances,
		Positions: positions,
	}

	ctx.Send(e.apiActorPID, portfolioMsg)

	balanceCount := 0
	positionCount := 0
	if balances != nil {
		balanceCount = len(balances)
	}
	if positions != nil {
		positionCount = len(positions)
	}

	e.logger.Debug().
		Str("exchange", e.exchangeName).
		Int("balance_count", balanceCount).
		Int("position_count", positionCount).
		Msg("Sent portfolio data to API actor")
}
