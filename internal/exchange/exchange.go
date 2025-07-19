package exchange

import (
	"context"
	"fmt"
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
	ConnectMessage       struct{}
	DisconnectMessage    struct{}
	SubscribeKlinesMsg   struct{ Symbols []string; Interval string }
	SubscribeOrderBookMsg struct{ Symbols []string }
	GetBalancesMsg       struct{}
	GetPositionsMsg      struct{}
	StatusMsg            struct{}
	
	// Data messages
	KlineDataMsg     struct{ Kline *exchanges.Kline }
	OrderBookDataMsg struct{ OrderBook *exchanges.OrderBook }
	OrderUpdateMsg   struct{ Order *exchanges.Order }
	PositionUpdateMsg struct{ Position *exchanges.Position }
	BalanceUpdateMsg struct{ Balance *exchanges.Balance }
)

// ExchangeActor manages exchange connections and child actors
type ExchangeActor struct {
	exchangeName   string
	config         *config.Config
	db             *database.DB
	logger         zerolog.Logger
	exchange       exchanges.Exchange
	factory        *exchanges.Factory
	
	// Child actors
	strategyActors  map[string]*actor.PID
	orderManagerPID *actor.PID
	riskManagerPID  *actor.PID
	portfolioPID    *actor.PID
	settingsPID     *actor.PID
	
	// State
	connected       bool
	subscribedKlines map[string]bool
	subscribedOrderBooks map[string]bool
}

// New creates a new exchange actor
func New(exchangeName string, exchangeConfig map[string]interface{}, cfg *config.Config, db *database.DB, logger zerolog.Logger) *ExchangeActor {
	factory := exchanges.NewFactory(logger)
	
	return &ExchangeActor{
		exchangeName:         exchangeName,
		config:              cfg,
		db:                  db,
		logger:              logger,
		factory:             factory,
		strategyActors:      make(map[string]*actor.PID),
		subscribedKlines:    make(map[string]bool),
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
	case StatusMsg:
		e.onStatus(ctx)
	case KlineDataMsg:
		e.onKlineData(ctx, msg)
	case OrderBookDataMsg:
		e.onOrderBookData(ctx, msg)
	default:
		e.logger.Warn().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received unknown message")
	}
}

func (e *ExchangeActor) onStarted(ctx *actor.Context) {
	e.logger.Info().Str("exchange", e.exchangeName).Msg("Exchange actor started")
	
	// Start child actors
	e.startChildActors(ctx)
	
	// Auto-connect to exchange
	ctx.Send(ctx.PID(), ConnectMessage{})
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
		return order.New(e.exchangeName, e.config, e.db, e.logger.With().Str("actor", "order_manager").Logger())
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
		"api_key": "",  // These would come from config
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
	
	// Note: This would use WebSocket in a full implementation
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
	
	for _, symbol := range msg.Symbols {
		e.subscribedOrderBooks[symbol] = true
	}
}

func (e *ExchangeActor) onGetBalances(ctx *actor.Context) {
	if !e.connected {
		e.logger.Error().Msg("Cannot get balances: not connected")
		ctx.Respond(fmt.Errorf("not connected"))
		return
	}
	
	balanceCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	balances, err := e.exchange.GetBalances(balanceCtx)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to get balances")
		ctx.Respond(err)
		return
	}
	
	ctx.Respond(balances)
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
	
	ctx.Respond(positions)
}

func (e *ExchangeActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":              e.exchangeName,
		"connected":            e.connected,
		"subscribed_klines":    len(e.subscribedKlines),
		"subscribed_orderbooks": len(e.subscribedOrderBooks),
		"strategy_actors":      len(e.strategyActors),
		"timestamp":            time.Now(),
	}
	
	ctx.Respond(status)
}

func (e *ExchangeActor) onKlineData(ctx *actor.Context, msg KlineDataMsg) {
	// Forward kline data to strategy actors
	for _, strategyPID := range e.strategyActors {
		ctx.Send(strategyPID, msg)
	}
}

func (e *ExchangeActor) onOrderBookData(ctx *actor.Context, msg OrderBookDataMsg) {
	// Forward order book data to strategy actors
	for _, strategyPID := range e.strategyActors {
		ctx.Send(strategyPID, msg)
	}
}

// StartStrategy starts a new strategy actor for a symbol
func (e *ExchangeActor) StartStrategy(ctx *actor.Context, strategyName, symbol string, config map[string]interface{}) error {
	strategyKey := fmt.Sprintf("%s:%s", strategyName, symbol)
	
	if _, exists := e.strategyActors[strategyKey]; exists {
		return fmt.Errorf("strategy %s already running for symbol %s", strategyName, symbol)
	}
	
	strategyPID := ctx.SpawnChild(func() actor.Receiver {
		return strategy.New(
			strategyName,
			symbol,
			e.exchangeName,
			config,
			e.config,
			e.db,
			e.logger.With().Str("actor", "strategy").Str("strategy", strategyName).Str("symbol", symbol).Logger(),
		)
	}, strategyKey)
	
	e.strategyActors[strategyKey] = strategyPID
	e.logger.Info().
		Str("strategy", strategyName).
		Str("symbol", symbol).
		Msg("Strategy actor started")
	
	return nil
}