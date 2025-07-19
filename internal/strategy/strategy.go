package strategy

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

// Messages for strategy actor communication
type (
	StartStrategyMsg   struct{}
	StopStrategyMsg    struct{}
	KlineDataMsg       struct{ Kline *exchanges.Kline }
	OrderBookDataMsg   struct{ OrderBook *exchanges.OrderBook }
	StatusMsg          struct{}
	ExecuteStrategyMsg struct{}
)

// StrategyActor executes trading strategies using Starlark
type StrategyActor struct {
	strategyName string
	symbol       string
	exchangeName string
	config       map[string]interface{}
	appConfig    *config.Config
	db           *database.DB
	logger       zerolog.Logger
	running      bool

	// Strategy execution
	engine      *StrategyEngine
	klineBuffer []*KlineData
	orderBook   *exchanges.OrderBook

	// Parent actor references
	orderManagerPID *actor.PID
	riskManagerPID  *actor.PID
}

// New creates a new strategy actor
func New(strategyName, symbol, exchangeName string, strategyConfig map[string]interface{}, appConfig *config.Config, db *database.DB, logger zerolog.Logger) *StrategyActor {
	return &StrategyActor{
		strategyName: strategyName,
		symbol:       symbol,
		exchangeName: exchangeName,
		config:       strategyConfig,
		appConfig:    appConfig,
		db:           db,
		logger:       logger,
		engine:       NewStrategyEngine(logger),
		klineBuffer:  make([]*KlineData, 0, 100), // Keep last 100 klines
	}
}

// SetParentActors sets references to parent actors for communication
func (s *StrategyActor) SetParentActors(orderManagerPID, riskManagerPID *actor.PID) {
	s.orderManagerPID = orderManagerPID
	s.riskManagerPID = riskManagerPID
}

// Receive handles incoming messages
func (s *StrategyActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		s.onStarted(ctx)
	case actor.Stopped:
		s.onStopped(ctx)
	case StartStrategyMsg:
		s.onStartStrategy(ctx)
	case StopStrategyMsg:
		s.onStopStrategy(ctx)
	case KlineDataMsg:
		s.onKlineData(ctx, msg)
	case OrderBookDataMsg:
		s.onOrderBookData(ctx, msg)
	case ExecuteStrategyMsg:
		s.onExecuteStrategy(ctx)
	case StatusMsg:
		s.onStatus(ctx)
	default:
		s.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (s *StrategyActor) onStarted(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor started")

	// Auto-start the strategy
	ctx.Send(ctx.PID(), StartStrategyMsg{})
}

func (s *StrategyActor) onStopped(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor stopped")
}

func (s *StrategyActor) onStartStrategy(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Starting strategy execution")

	s.running = true

	// Start periodic strategy execution (every 30 seconds)
	ctx.SendRepeat(ctx.PID(), ExecuteStrategyMsg{}, 30*time.Second)
}

func (s *StrategyActor) onStopStrategy(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Stopping strategy execution")

	s.running = false
}

func (s *StrategyActor) onKlineData(ctx *actor.Context, msg KlineDataMsg) {
	if !s.running {
		return
	}

	// Check if this kline matches the strategy's configured interval
	configuredInterval := s.appConfig.Strategies.DefaultInterval
	if strategyInterval, exists := s.config["interval"]; exists {
		if intervalStr, ok := strategyInterval.(string); ok {
			configuredInterval = intervalStr
		}
	}

	// Only process klines that match our configured interval
	if msg.Kline.Interval != configuredInterval {
		s.logger.Debug().
			Str("symbol", msg.Kline.Symbol).
			Str("received_interval", msg.Kline.Interval).
			Str("configured_interval", configuredInterval).
			Msg("Ignoring kline data - interval mismatch")
		return
	}

	// Convert exchange kline to strategy kline
	klineData := &KlineData{
		Timestamp: msg.Kline.Timestamp,
		Open:      msg.Kline.Open,
		High:      msg.Kline.High,
		Low:       msg.Kline.Low,
		Close:     msg.Kline.Close,
		Volume:    msg.Kline.Volume,
	}

	// Add to buffer
	s.klineBuffer = append(s.klineBuffer, klineData)

	// Keep only last 100 klines
	if len(s.klineBuffer) > 100 {
		s.klineBuffer = s.klineBuffer[1:]
	}

	s.logger.Debug().
		Str("symbol", msg.Kline.Symbol).
		Str("interval", msg.Kline.Interval).
		Time("timestamp", msg.Kline.Timestamp).
		Float64("close", msg.Kline.Close).
		Msg("Processing kline data for strategy")

	// Trigger strategy execution if we have enough data
	if len(s.klineBuffer) >= 26 { // Need at least 26 for most indicators
		ctx.Send(ctx.PID(), ExecuteStrategyMsg{})
	}
}

func (s *StrategyActor) onOrderBookData(ctx *actor.Context, msg OrderBookDataMsg) {
	if !s.running {
		return
	}

	s.orderBook = msg.OrderBook

	s.logger.Debug().
		Str("symbol", msg.OrderBook.Symbol).
		Float64("bid", msg.OrderBook.Bids[0].Price).
		Float64("ask", msg.OrderBook.Asks[0].Price).
		Msg("Received order book data")
}

func (s *StrategyActor) onExecuteStrategy(ctx *actor.Context) {
	if !s.running || len(s.klineBuffer) < 26 {
		return
	}

	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Executing strategy")

	// Prepare strategy context
	strategyCtx := &StrategyContext{
		Symbol:    s.symbol,
		Exchange:  s.exchangeName,
		Klines:    s.klineBuffer,
		OrderBook: s.orderBook,
		Config:    s.config,
		// TODO: Add balances, positions, open orders from exchange
	}

	// Execute strategy
	signal, err := s.engine.ExecuteStrategy(s.strategyName, strategyCtx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Strategy execution failed")
		return
	}

	// Process signal
	if signal.Action != "hold" {
		s.logger.Info().
			Str("action", signal.Action).
			Float64("quantity", signal.Quantity).
			Float64("price", signal.Price).
			Str("reason", signal.Reason).
			Msg("Strategy generated signal")

		// Send order to order manager (if we have reference)
		if s.orderManagerPID != nil {
			orderRequest := map[string]interface{}{
				"symbol":   s.symbol,
				"side":     signal.Action,
				"type":     signal.Type,
				"quantity": signal.Quantity,
				"price":    signal.Price,
				"reason":   signal.Reason,
			}
			ctx.Send(s.orderManagerPID, orderRequest)
		}

		// Notify risk manager (if we have reference)
		if s.riskManagerPID != nil {
			riskCheck := map[string]interface{}{
				"symbol":   s.symbol,
				"action":   signal.Action,
				"quantity": signal.Quantity,
				"price":    signal.Price,
			}
			ctx.Send(s.riskManagerPID, riskCheck)
		}
	}
}

func (s *StrategyActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"strategy_name":   s.strategyName,
		"symbol":          s.symbol,
		"exchange":        s.exchangeName,
		"running":         s.running,
		"klines_buffered": len(s.klineBuffer),
		"has_orderbook":   s.orderBook != nil,
		"timestamp":       time.Now(),
	}

	ctx.Respond(status)
}
