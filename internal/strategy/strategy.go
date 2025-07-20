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
	GetLogsMsg         struct{ Limit int }
	LogsResponseMsg    struct{ Logs []StrategyLog }
	// Message sent from strategy to exchange actor to register subscription preferences
	StrategySubscriptionMsg struct {
		Symbol   string
		Interval string
	}
)

// StrategyLog represents a log entry from strategy execution
type StrategyLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

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
	initialized  bool   // Whether strategy has been initialized with historical data
	interval     string // Interval from strategy script

	// Strategy execution
	engine      *StrategyEngine
	klineBuffer []*KlineData
	orderBook   *exchanges.OrderBook
	callbacks   *StrategyCallbacks // Cache of available callbacks

	// Parent actor references
	orderManagerPID *actor.PID
	riskManagerPID  *actor.PID
	exchangePID     *actor.PID // Reference to parent exchange actor

	// Log storage (in-memory circular buffer)
	logs    []StrategyLog
	maxLogs int
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
		logs:         make([]StrategyLog, 0),     // Initialize logs slice
		maxLogs:      100,                        // Keep last 100 log entries
	}
}

// SetParentActors sets references to parent actors for communication
func (s *StrategyActor) SetParentActors(orderManagerPID, riskManagerPID, exchangePID *actor.PID) {
	s.orderManagerPID = orderManagerPID
	s.riskManagerPID = riskManagerPID
	s.exchangePID = exchangePID
}

// Receive handles incoming messages
func (s *StrategyActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		s.onStarted(ctx)
	case actor.Stopped:
		s.onStopped(ctx)
	case actor.Initialized:
		s.onInitialized(ctx)
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
	case GetLogsMsg:
		s.onGetLogs(ctx, msg)
	default:
		// Reduced chattiness - only log unknown message types occasionally
		s.logger.Info().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received unknown message")
	}
}

func (s *StrategyActor) onStarted(ctx *actor.Context) {
	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor started")

	// Set strategy actor reference in engine for logging
	s.engine.SetStrategyActor(s)

	// Auto-start the strategy
	ctx.Send(ctx.PID(), StartStrategyMsg{})
}

func (s *StrategyActor) onInitialized(ctx *actor.Context) {
	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor initialized")
}

func (s *StrategyActor) onStopped(ctx *actor.Context) {
	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor stopped")
}

func (s *StrategyActor) onStartStrategy(ctx *actor.Context) {
	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Starting strategy execution")

	// Initialize strategy engine if not already done
	if s.engine == nil {
		s.engine = NewStrategyEngine(s.logger)
	}

	// Validate which callbacks are available in the strategy
	callbacks, err := s.engine.ValidateCallbacks(s.strategyName)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to validate strategy callbacks")
		s.addLog("error", fmt.Sprintf("Failed to validate strategy callbacks: %v", err), nil)
		return
	}
	s.callbacks = callbacks

	s.logger.Debug().
		Str("strategy", s.strategyName).
		Bool("has_on_kline", callbacks.HasOnKline).
		Bool("has_on_orderbook", callbacks.HasOnOrderBook).
		Bool("has_on_ticker", callbacks.HasOnTicker).
		Bool("has_on_start", callbacks.HasOnStart).
		Bool("has_on_stop", callbacks.HasOnStop).
		Msg("Strategy callbacks validated")

	// Extract interval from strategy script
	interval, err := s.engine.GetStrategyInterval(s.strategyName)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get strategy interval, using default")
		s.addLog("warning", fmt.Sprintf("Failed to get strategy interval, using default: %v", err), nil)
		interval = "1m" // Default fallback
	}
	s.interval = interval

	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Str("interval", s.interval).
		Msg("Strategy interval extracted from script")

	s.addLog("info", fmt.Sprintf("Strategy %s initializing for %s", s.strategyName, s.symbol), map[string]interface{}{
		"strategy": s.strategyName,
		"symbol":   s.symbol,
		"interval": s.interval,
	})

	// Call on_start callback if available
	if callbacks.HasOnStart {
		strategyCtx := &StrategyContext{
			Symbol:    s.symbol,
			Exchange:  s.exchangeName,
			Klines:    s.klineBuffer,
			OrderBook: s.orderBook,
			Config:    s.config,
			// TODO: Add balances, positions, open orders from exchange
		}
		err := s.engine.ExecuteStartCallback(s.strategyName, strategyCtx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to execute on_start callback")
		}
	}

	// Register subscription with exchange actor for efficient routing
	if s.exchangePID != nil {
		ctx.Send(s.exchangePID, StrategySubscriptionMsg{
			Symbol:   s.symbol,
			Interval: s.interval,
		})
		s.logger.Debug().
			Str("symbol", s.symbol).
			Str("interval", s.interval).
			Msg("Registered strategy subscription with exchange")
	}

	// Fetch historical klines to populate initial data before starting
	s.fetchHistoricalKlines(ctx)

	// Strategy will be marked as running and periodic execution will start
	// after historical data is received (handled in onKlineData when we reach minimum buffer size)
}

func (s *StrategyActor) onStopStrategy(ctx *actor.Context) {
	s.logger.Debug().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Stopping strategy execution")

	s.addLog("info", fmt.Sprintf("Strategy %s stopped for %s", s.strategyName, s.symbol), map[string]interface{}{
		"strategy": s.strategyName,
		"symbol":   s.symbol,
	})

	// Call on_stop callback if available
	if s.callbacks != nil && s.callbacks.HasOnStop {
		strategyCtx := &StrategyContext{
			Symbol:    s.symbol,
			Exchange:  s.exchangeName,
			Klines:    s.klineBuffer,
			OrderBook: s.orderBook,
			Config:    s.config,
			// TODO: Add balances, positions, open orders from exchange
		}
		err := s.engine.ExecuteStopCallback(s.strategyName, strategyCtx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to execute on_stop callback")
			s.addLog("error", fmt.Sprintf("Failed to execute on_stop callback: %v", err), nil)
		}
	}

	s.running = false
}

func (s *StrategyActor) onKlineData(ctx *actor.Context, msg KlineDataMsg) {
	// Always process klines that match our strategy's symbol and interval
	if msg.Kline.Symbol != s.symbol || msg.Kline.Interval != s.interval {
		// Reduced chattiness - don't log mismatches
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
		Int("buffer_size", len(s.klineBuffer)).
		Bool("running", s.running).
		Bool("initialized", s.initialized).
		Msg("Processing kline data for strategy")

	// If this is the first time we have sufficient historical data, start the strategy
	if !s.initialized && len(s.klineBuffer) >= 10 { // Require at least 10 klines before starting
		s.initialized = true
		s.running = true

		s.logger.Info().
			Str("strategy", s.strategyName).
			Str("symbol", s.symbol).
			Str("interval", s.interval).
			Int("historical_klines", len(s.klineBuffer)).
			Msg("Strategy initialized with historical data and started")

		s.addLog("info", fmt.Sprintf("Strategy %s started with %d historical klines for %s %s", s.strategyName, len(s.klineBuffer), s.symbol, s.interval), map[string]interface{}{
			"strategy":         s.strategyName,
			"symbol":           s.symbol,
			"interval":         s.interval,
			"historical_count": len(s.klineBuffer),
		})

		// Start periodic strategy execution (every 30 seconds)
		ctx.SendRepeat(ctx.PID(), ExecuteStrategyMsg{}, 30*time.Second)
	}

	// Only process real-time klines for trading signals if strategy is fully running
	if !s.running {
		return
	}

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
	if len(s.klineBuffer) >= 1 {
		s.executeKlineCallback(ctx, msg.Kline)
	}
}

func (s *StrategyActor) onOrderBookData(ctx *actor.Context, msg OrderBookDataMsg) {
	if !s.running {
		return
	}

	s.orderBook = msg.OrderBook

	// Check if order book has valid bids and asks before accessing
	if len(msg.OrderBook.Bids) == 0 || len(msg.OrderBook.Asks) == 0 {
		s.logger.Debug().
			Str("symbol", msg.OrderBook.Symbol).
			Int("bids", len(msg.OrderBook.Bids)).
			Int("asks", len(msg.OrderBook.Asks)).
			Msg("Received empty order book, skipping")
		return
	}

	s.logger.Debug().
		Str("symbol", msg.OrderBook.Symbol).
		Float64("bid", msg.OrderBook.Bids[0].Price).
		Float64("ask", msg.OrderBook.Asks[0].Price).
		Msg("Received order book data")

	// Send price update to order manager for stop/trailing orders
	if s.orderManagerPID != nil {
		midPrice := (msg.OrderBook.Bids[0].Price + msg.OrderBook.Asks[0].Price) / 2
		priceUpdate := map[string]interface{}{
			"type":   "price_update",
			"symbol": msg.OrderBook.Symbol,
			"price":  midPrice,
		}
		ctx.Send(s.orderManagerPID, priceUpdate)
	}

	// Execute strategy with orderbook callback if we have enough data
	if len(s.klineBuffer) >= 1 {
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
	if len(s.klineBuffer) >= 1 {
		s.executeTickerCallback(ctx, msg.Ticker)
	}
}

func (s *StrategyActor) onExecuteStrategy(ctx *actor.Context) {
	if !s.running || len(s.klineBuffer) < 1 {
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
		s.addLog("error", fmt.Sprintf("Strategy execution failed: %v", err), nil)
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

		s.addLog("info", fmt.Sprintf("Generated %s signal", signal.Action), map[string]interface{}{
			"action":   signal.Action,
			"quantity": signal.Quantity,
			"price":    signal.Price,
			"reason":   signal.Reason,
		})

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

	// Check if on_kline callback exists before executing
	if s.callbacks == nil || !s.callbacks.HasOnKline {
		s.logger.Debug().Msg("Skipping kline callback - not implemented in strategy")
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

	// Check if on_orderbook callback exists before executing
	if s.callbacks == nil || !s.callbacks.HasOnOrderBook {
		s.logger.Debug().Msg("Skipping orderbook callback - not implemented in strategy")
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

	// Check if on_ticker callback exists before executing
	if s.callbacks == nil || !s.callbacks.HasOnTicker {
		s.logger.Debug().Msg("Skipping ticker callback - not implemented in strategy")
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

// onGetLogs handles requests for strategy logs
func (s *StrategyActor) onGetLogs(ctx *actor.Context, msg GetLogsMsg) {
	limit := msg.Limit
	if limit <= 0 || limit > len(s.logs) {
		limit = len(s.logs)
	}

	// Return the most recent logs (last 'limit' entries)
	start := len(s.logs) - limit
	if start < 0 {
		start = 0
	}

	logs := make([]StrategyLog, limit)
	copy(logs, s.logs[start:])

	ctx.Respond(LogsResponseMsg{Logs: logs})
}

// addLog adds a log entry to the strategy's log buffer
func (s *StrategyActor) addLog(level, message string, context map[string]interface{}) {
	log := StrategyLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   context,
	}

	// Add to logs slice
	s.logs = append(s.logs, log)

	// Maintain circular buffer by removing old entries
	if len(s.logs) > s.maxLogs {
		// Keep the most recent maxLogs entries
		s.logs = s.logs[len(s.logs)-s.maxLogs:]
	}
}

// fetchHistoricalKlines fetches historical kline data from the exchange to populate initial strategy data
func (s *StrategyActor) fetchHistoricalKlines(ctx *actor.Context) {
	if s.exchangePID == nil {
		s.logger.Warn().Msg("No exchange actor reference, skipping historical klines fetch")
		return
	}

	// Import the message type from exchange package - we need to use a map to avoid import cycle
	fetchMsg := map[string]interface{}{
		"type":     "fetch_historical_klines",
		"symbol":   s.symbol,
		"interval": s.interval,
		"limit":    100,
	}

	// Request last 100 klines for this symbol/interval
	ctx.Send(s.exchangePID, fetchMsg)

	s.logger.Debug().
		Str("symbol", s.symbol).
		Str("interval", s.interval).
		Int("limit", 100).
		Msg("Requested historical klines from exchange")

	s.addLog("info", fmt.Sprintf("Fetching historical data for %s %s", s.symbol, s.interval), map[string]interface{}{
		"symbol":   s.symbol,
		"interval": s.interval,
		"limit":    100,
	})
}
