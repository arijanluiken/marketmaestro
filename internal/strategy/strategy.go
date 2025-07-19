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
	TickerDataMsg      struct{ Ticker *exchanges.Ticker }
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
	interval     string // Interval from strategy script

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
	case TickerDataMsg:
		s.onTickerData(ctx, msg)
	case ExecuteStrategyMsg:
		s.onExecuteStrategy(ctx)
	case StatusMsg:
		s.onStatus(ctx)
	default:
		// Reduced chattiness - only log unknown message types occasionally
		s.logger.Info().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received unknown message")
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

	// Initialize strategy engine if not already done
	if s.engine == nil {
		s.engine = NewStrategyEngine(s.logger)
	}

	// Extract interval from strategy script
	interval, err := s.engine.GetStrategyInterval(s.strategyName)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get strategy interval, using default")
		interval = "1m" // Default fallback
	}
	s.interval = interval

	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Str("interval", s.interval).
		Msg("Strategy interval extracted from script")

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

	// Only process klines that match our strategy's interval
	if msg.Kline.Interval != s.interval {
		// Reduced chattiness - don't log interval mismatches
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

	// Send price update to order manager
	if s.orderManagerPID != nil {
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": msg.Kline.Symbol,
			"price":  msg.Kline.Close,
		}
		ctx.Send(s.orderManagerPID, priceUpdate)
	}

	// Trigger strategy execution with kline callback if we have enough data
	if len(s.klineBuffer) >= 26 {
		s.executeKlineCallback(ctx, msg.Kline)
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

	// Send price update to order manager for stop/trailing orders
	if s.orderManagerPID != nil && len(msg.OrderBook.Bids) > 0 {
		midPrice := (msg.OrderBook.Bids[0].Price + msg.OrderBook.Asks[0].Price) / 2
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": msg.OrderBook.Symbol,
			"price":  midPrice,
		}
		ctx.Send(s.orderManagerPID, priceUpdate)
	}

	// Execute strategy with orderbook callback if we have enough data
	if len(s.klineBuffer) >= 26 {
		s.executeOrderBookCallback(ctx, msg.OrderBook)
	}
}

func (s *StrategyActor) onTickerData(ctx *actor.Context, msg TickerDataMsg) {
	if !s.running {
		return
	}

	s.logger.Debug().
		Str("symbol", msg.Ticker.Symbol).
		Float64("price", msg.Ticker.Price).
		Float64("volume", msg.Ticker.Volume).
		Msg("Received ticker data")

	// Send price update to order manager
	if s.orderManagerPID != nil {
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": msg.Ticker.Symbol,
			"price":  msg.Ticker.Price,
		}
		ctx.Send(s.orderManagerPID, priceUpdate)
	}

	// Execute strategy with ticker callback if we have enough data
	if len(s.klineBuffer) >= 26 {
		s.executeTickerCallback(ctx, msg.Ticker)
	}
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

// executeKlineCallback executes the strategy using the on_kline callback
func (s *StrategyActor) executeKlineCallback(ctx *actor.Context, kline *exchanges.Kline) {
	if !s.running {
		return
	}

	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Executing strategy with kline callback")

	// Prepare strategy context
	strategyCtx := &StrategyContext{
		Symbol:    s.symbol,
		Exchange:  s.exchangeName,
		Klines:    s.klineBuffer,
		OrderBook: s.orderBook,
		Config:    s.config,
		// TODO: Add balances, positions, open orders from exchange
	}

	// Execute strategy with kline callback
	signal, err := s.engine.ExecuteKlineCallback(s.strategyName, strategyCtx, kline)
	if err != nil {
		s.logger.Error().Err(err).Msg("Strategy kline callback execution failed")
		return
	}

	s.processStrategySignal(ctx, signal, "kline_callback")
}

// executeOrderBookCallback executes the strategy using the on_orderbook callback
func (s *StrategyActor) executeOrderBookCallback(ctx *actor.Context, orderBook *exchanges.OrderBook) {
	if !s.running {
		return
	}

	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Executing strategy with orderbook callback")

	// Prepare strategy context
	strategyCtx := &StrategyContext{
		Symbol:    s.symbol,
		Exchange:  s.exchangeName,
		Klines:    s.klineBuffer,
		OrderBook: s.orderBook,
		Config:    s.config,
		// TODO: Add balances, positions, open orders from exchange
	}

	// Execute strategy with orderbook callback
	signal, err := s.engine.ExecuteOrderBookCallback(s.strategyName, strategyCtx, orderBook)
	if err != nil {
		s.logger.Error().Err(err).Msg("Strategy orderbook callback execution failed")
		return
	}

	s.processStrategySignal(ctx, signal, "orderbook_callback")
}

// executeTickerCallback executes the strategy using the on_ticker callback
func (s *StrategyActor) executeTickerCallback(ctx *actor.Context, ticker *exchanges.Ticker) {
	if !s.running {
		return
	}

	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Executing strategy with ticker callback")

	// Prepare strategy context
	strategyCtx := &StrategyContext{
		Symbol:    s.symbol,
		Exchange:  s.exchangeName,
		Klines:    s.klineBuffer,
		OrderBook: s.orderBook,
		Config:    s.config,
		// TODO: Add balances, positions, open orders from exchange
	}

	// Execute strategy with ticker callback
	signal, err := s.engine.ExecuteTickerCallback(s.strategyName, strategyCtx, ticker)
	if err != nil {
		s.logger.Error().Err(err).Msg("Strategy ticker callback execution failed")
		return
	}

	s.processStrategySignal(ctx, signal, "ticker_callback")
}

// processStrategySignal processes a strategy signal and sends orders to the order manager
func (s *StrategyActor) processStrategySignal(ctx *actor.Context, signal *StrategySignal, source string) {
	if signal.Action != "hold" {
		s.logger.Info().
			Str("action", signal.Action).
			Float64("quantity", signal.Quantity).
			Float64("price", signal.Price).
			Str("type", signal.Type).
			Str("reason", signal.Reason).
			Str("source", source).
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
				"source":   source,
			}
			ctx.Send(s.riskManagerPID, riskCheck)
		}
	}
}
