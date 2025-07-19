package strategy

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

// TechnicalIndicators provides common trading indicators
type TechnicalIndicators struct {
	logger zerolog.Logger
}

// StrategyEngine executes Starlark-based trading strategies
type StrategyEngine struct {
	logger      zerolog.Logger
	indicators  *TechnicalIndicators
	builtin     starlark.StringDict
	scriptCache map[string]*starlark.Program
}

// KlineData represents historical price data for strategies
type KlineData struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// StrategyContext provides runtime context for strategy execution
type StrategyContext struct {
	Symbol     string
	Exchange   string
	Klines     []*KlineData
	OrderBook  *exchanges.OrderBook
	Config     map[string]interface{}
	Balances   []*exchanges.Balance
	Positions  []*exchanges.Position
	OpenOrders []*exchanges.Order
}

// StrategySignal represents a trading signal from a strategy
type StrategySignal struct {
	Action   string // "buy", "sell", "hold"
	Quantity float64
	Price    float64
	Type     string // "market", "limit"
	Reason   string
}

// NewStrategyEngine creates a new strategy engine
func NewStrategyEngine(logger zerolog.Logger) *StrategyEngine {
	indicators := &TechnicalIndicators{logger: logger}

	engine := &StrategyEngine{
		logger:      logger,
		indicators:  indicators,
		scriptCache: make(map[string]*starlark.Program),
	}

	engine.setupBuiltins()
	return engine
}

func (se *StrategyEngine) setupBuiltins() {
	// Add built-in functions
	se.builtin = starlark.StringDict{
		"print": starlark.NewBuiltin("print", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			fmt.Println(args.String())
			return starlark.None, nil
		}),
		"len": starlark.NewBuiltin("len", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("len() takes exactly one argument")
			}
			return starlark.MakeInt(starlark.Len(args[0])), nil
		}),
		"range": starlark.NewBuiltin("range", se.starlarkBuiltinRange),
		"math": starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
			"abs": starlark.NewBuiltin("abs", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("abs() takes exactly one argument")
				}
				if x, ok := args[0].(starlark.Float); ok {
					return starlark.Float(math.Abs(float64(x))), nil
				}
				if x, ok := args[0].(starlark.Int); ok {
					val, _ := x.Int64()
					if val < 0 {
						return starlark.MakeInt64(-val), nil
					}
					return x, nil
				}
				return nil, fmt.Errorf("abs() requires a number")
			}),
		}),
	}
}

func (se *StrategyEngine) starlarkBuiltinRange(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var start, stop, step int64 = 0, 0, 1

	switch len(args) {
	case 1:
		if x, ok := args[0].(starlark.Int); ok {
			stop, _ = x.Int64()
		} else {
			return nil, fmt.Errorf("range() argument must be an integer")
		}
	case 2:
		if x, ok := args[0].(starlark.Int); ok {
			start, _ = x.Int64()
		} else {
			return nil, fmt.Errorf("range() start must be an integer")
		}
		if x, ok := args[1].(starlark.Int); ok {
			stop, _ = x.Int64()
		} else {
			return nil, fmt.Errorf("range() stop must be an integer")
		}
	case 3:
		if x, ok := args[0].(starlark.Int); ok {
			start, _ = x.Int64()
		} else {
			return nil, fmt.Errorf("range() start must be an integer")
		}
		if x, ok := args[1].(starlark.Int); ok {
			stop, _ = x.Int64()
		} else {
			return nil, fmt.Errorf("range() stop must be an integer")
		}
		if x, ok := args[2].(starlark.Int); ok {
			step, _ = x.Int64()
		} else {
			return nil, fmt.Errorf("range() step must be an integer")
		}
		if step == 0 {
			return nil, fmt.Errorf("range() step cannot be zero")
		}
	default:
		return nil, fmt.Errorf("range() takes 1 to 3 arguments")
	}

	var result []starlark.Value
	if step > 0 {
		for i := start; i < stop; i += step {
			result = append(result, starlark.MakeInt64(i))
		}
	} else {
		for i := start; i > stop; i += step {
			result = append(result, starlark.MakeInt64(i))
		}
	}

	return starlark.NewList(result), nil
}

// ExecuteStrategy runs a strategy script with the given context
func (se *StrategyEngine) ExecuteStrategy(strategyName string, ctx *StrategyContext) (*StrategySignal, error) {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s", strategyName),
	}

	// Prepare globals with context data
	globals := se.prepareGlobals(ctx)

	// Execute strategy
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return nil, fmt.Errorf("strategy execution failed: %w", err)
	}

	// Extract signal from result
	signal, err := se.extractSignal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to extract signal: %w", err)
	}

	se.logger.Debug().
		Str("strategy", strategyName).
		Str("action", signal.Action).
		Float64("quantity", signal.Quantity).
		Str("reason", signal.Reason).
		Msg("Strategy executed")

	return signal, nil
}

// loadStrategy loads and compiles a strategy script
func (se *StrategyEngine) loadStrategy(name string) (string, error) {
	// First try strategy directory
	strategyPath := filepath.Join("strategy", fmt.Sprintf("%s.star", name))
	if data, err := os.ReadFile(strategyPath); err == nil {
		return string(data), nil
	}

	// Then try root directory
	rootPath := fmt.Sprintf("%s.star", name)
	if data, err := os.ReadFile(rootPath); err == nil {
		return string(data), nil
	}

	// Then try external file
	externalPath := filepath.Join("strategies", fmt.Sprintf("%s.star", name))
	if data, err := os.ReadFile(externalPath); err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("strategy script not found: %s", name)
}

// prepareGlobals creates the global variables for strategy execution
func (se *StrategyEngine) prepareGlobals(ctx *StrategyContext) starlark.StringDict {
	globals := make(starlark.StringDict)

	// Copy builtins
	for k, v := range se.builtin {
		globals[k] = v
	}

	// Add context data
	globals["symbol"] = starlark.String(ctx.Symbol)
	globals["exchange"] = starlark.String(ctx.Exchange)
	globals["config"] = se.mapToStarlark(ctx.Config)

	// Add kline data
	if len(ctx.Klines) > 0 {
		globals["klines"] = se.klinesToStarlark(ctx.Klines)
		globals["close"] = se.extractPrices(ctx.Klines, "close")
		globals["open"] = se.extractPrices(ctx.Klines, "open")
		globals["high"] = se.extractPrices(ctx.Klines, "high")
		globals["low"] = se.extractPrices(ctx.Klines, "low")
		globals["volume"] = se.extractPrices(ctx.Klines, "volume")
	}

	// Add order book data
	if ctx.OrderBook != nil {
		globals["bid"] = starlark.Float(ctx.OrderBook.Bids[0].Price)
		globals["ask"] = starlark.Float(ctx.OrderBook.Asks[0].Price)
		globals["spread"] = starlark.Float(ctx.OrderBook.Asks[0].Price - ctx.OrderBook.Bids[0].Price)
	}

	return globals
}

// extractSignal extracts trading signal from strategy execution result
func (se *StrategyEngine) extractSignal(result starlark.StringDict) (*StrategySignal, error) {
	signal := &StrategySignal{
		Action: "hold",
		Type:   "market",
	}

	if action, ok := result["action"]; ok {
		if s, ok := action.(starlark.String); ok {
			signal.Action = string(s)
		}
	}

	if quantity, ok := result["quantity"]; ok {
		if f, ok := quantity.(starlark.Float); ok {
			signal.Quantity = float64(f)
		}
	}

	if price, ok := result["price"]; ok {
		if f, ok := price.(starlark.Float); ok {
			signal.Price = float64(f)
		}
	}

	if orderType, ok := result["type"]; ok {
		if s, ok := orderType.(starlark.String); ok {
			signal.Type = string(s)
		}
	}

	if reason, ok := result["reason"]; ok {
		if s, ok := reason.(starlark.String); ok {
			signal.Reason = string(s)
		}
	}

	return signal, nil
}

// ExecuteKlineCallback runs the on_kline callback in a strategy script
func (se *StrategyEngine) ExecuteKlineCallback(strategyName string, ctx *StrategyContext, kline *exchanges.Kline) (*StrategySignal, error) {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-kline", strategyName),
	}

	// Prepare globals with context data
	globals := se.prepareGlobals(ctx)

	// Add kline object
	klineDict := starlark.NewDict(6)
	klineDict.SetKey(starlark.String("timestamp"), starlark.String(kline.Timestamp.Format("2006-01-02T15:04:05Z")))
	klineDict.SetKey(starlark.String("open"), starlark.Float(kline.Open))
	klineDict.SetKey(starlark.String("high"), starlark.Float(kline.High))
	klineDict.SetKey(starlark.String("low"), starlark.Float(kline.Low))
	klineDict.SetKey(starlark.String("close"), starlark.Float(kline.Close))
	klineDict.SetKey(starlark.String("volume"), starlark.Float(kline.Volume))
	globals["kline"] = klineDict

	// Execute strategy
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return nil, fmt.Errorf("strategy execution failed: %w", err)
	}

	// Check if on_kline function exists and call it
	if onKlineFn, ok := result["on_kline"]; ok {
		if fn, ok := onKlineFn.(*starlark.Function); ok {
			args := starlark.Tuple{globals["kline"]}
			signalResult, err := starlark.Call(thread, fn, args, nil)
			if err != nil {
				return nil, fmt.Errorf("on_kline callback failed: %w", err)
			}

			// Extract signal from callback result
			if signalDict, ok := signalResult.(*starlark.Dict); ok {
				return se.extractSignalFromDict(signalDict)
			}
		}
	}

	// Fallback to legacy execution if no callback
	return se.extractSignal(result)
}

// ExecuteOrderBookCallback runs the on_orderbook callback in a strategy script
func (se *StrategyEngine) ExecuteOrderBookCallback(strategyName string, ctx *StrategyContext, orderBook *exchanges.OrderBook) (*StrategySignal, error) {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-orderbook", strategyName),
	}

	// Prepare globals with context data
	globals := se.prepareGlobals(ctx)

	// Add orderbook object
	orderBookDict := starlark.NewDict(4)
	orderBookDict.SetKey(starlark.String("symbol"), starlark.String(orderBook.Symbol))
	orderBookDict.SetKey(starlark.String("timestamp"), starlark.String(orderBook.Timestamp.Format("2006-01-02T15:04:05Z")))

	// Add bids and asks
	bids := starlark.NewList([]starlark.Value{})
	for _, bid := range orderBook.Bids {
		bidDict := starlark.NewDict(2)
		bidDict.SetKey(starlark.String("price"), starlark.Float(bid.Price))
		bidDict.SetKey(starlark.String("quantity"), starlark.Float(bid.Quantity))
		bids.Append(bidDict)
	}

	asks := starlark.NewList([]starlark.Value{})
	for _, ask := range orderBook.Asks {
		askDict := starlark.NewDict(2)
		askDict.SetKey(starlark.String("price"), starlark.Float(ask.Price))
		askDict.SetKey(starlark.String("quantity"), starlark.Float(ask.Quantity))
		asks.Append(askDict)
	}

	orderBookDict.SetKey(starlark.String("bids"), bids)
	orderBookDict.SetKey(starlark.String("asks"), asks)
	globals["orderbook"] = orderBookDict

	// Execute strategy
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return nil, fmt.Errorf("strategy execution failed: %w", err)
	}

	// Check if on_orderbook function exists and call it
	if onOrderBookFn, ok := result["on_orderbook"]; ok {
		if fn, ok := onOrderBookFn.(*starlark.Function); ok {
			args := starlark.Tuple{globals["orderbook"]}
			signalResult, err := starlark.Call(thread, fn, args, nil)
			if err != nil {
				return nil, fmt.Errorf("on_orderbook callback failed: %w", err)
			}

			// Extract signal from callback result
			if signalDict, ok := signalResult.(*starlark.Dict); ok {
				return se.extractSignalFromDict(signalDict)
			}
		}
	}

	// Fallback to legacy execution if no callback
	return se.extractSignal(result)
}

// ExecuteTickerCallback runs the on_ticker callback in a strategy script
func (se *StrategyEngine) ExecuteTickerCallback(strategyName string, ctx *StrategyContext, ticker *exchanges.Ticker) (*StrategySignal, error) {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-ticker", strategyName),
	}

	// Prepare globals with context data
	globals := se.prepareGlobals(ctx)

	// Add ticker object
	tickerDict := starlark.NewDict(4)
	tickerDict.SetKey(starlark.String("symbol"), starlark.String(ticker.Symbol))
	tickerDict.SetKey(starlark.String("price"), starlark.Float(ticker.Price))
	tickerDict.SetKey(starlark.String("volume"), starlark.Float(ticker.Volume))
	tickerDict.SetKey(starlark.String("timestamp"), starlark.String(ticker.Timestamp.Format("2006-01-02T15:04:05Z")))
	globals["ticker"] = tickerDict

	// Execute strategy
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return nil, fmt.Errorf("strategy execution failed: %w", err)
	}

	// Check if on_ticker function exists and call it
	if onTickerFn, ok := result["on_ticker"]; ok {
		if fn, ok := onTickerFn.(*starlark.Function); ok {
			args := starlark.Tuple{globals["ticker"]}
			signalResult, err := starlark.Call(thread, fn, args, nil)
			if err != nil {
				return nil, fmt.Errorf("on_ticker callback failed: %w", err)
			}

			// Extract signal from callback result
			if signalDict, ok := signalResult.(*starlark.Dict); ok {
				return se.extractSignalFromDict(signalDict)
			}
		}
	}

	// Fallback to legacy execution if no callback
	return se.extractSignal(result)
}

// extractSignalFromDict extracts a trading signal from a Starlark dictionary
func (se *StrategyEngine) extractSignalFromDict(dict *starlark.Dict) (*StrategySignal, error) {
	signal := &StrategySignal{
		Action: "hold",
		Type:   "market",
	}

	if action, found, _ := dict.Get(starlark.String("action")); found {
		if s, ok := action.(starlark.String); ok {
			signal.Action = string(s)
		}
	}

	if quantity, found, _ := dict.Get(starlark.String("quantity")); found {
		if f, ok := quantity.(starlark.Float); ok {
			signal.Quantity = float64(f)
		}
	}

	if price, found, _ := dict.Get(starlark.String("price")); found {
		if f, ok := price.(starlark.Float); ok {
			signal.Price = float64(f)
		}
	}

	if orderType, found, _ := dict.Get(starlark.String("type")); found {
		if s, ok := orderType.(starlark.String); ok {
			signal.Type = string(s)
		}
	}

	if reason, found, _ := dict.Get(starlark.String("reason")); found {
		if s, ok := reason.(starlark.String); ok {
			signal.Reason = string(s)
		}
	}

	return signal, nil
}

// GetStrategyInterval extracts the interval from a strategy script
func (se *StrategyEngine) GetStrategyInterval(strategyName string) (string, error) {
	// Load strategy script
	scriptContent, err := se.loadStrategy(strategyName)
	if err != nil {
		return "", fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Simple text parsing approach to find interval in settings function
	lines := strings.Split(scriptContent, "\n")
	inSettingsFunction := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if we're entering the settings function
		if strings.HasPrefix(line, "def settings()") {
			inSettingsFunction = true
			continue
		}

		// Check if we're exiting the settings function
		if inSettingsFunction && strings.HasPrefix(line, "def ") {
			break
		}

		// Look for interval within settings function
		if inSettingsFunction && strings.Contains(line, "interval") && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				intervalPart := strings.TrimSpace(parts[1])
				// Remove quotes and comments
				intervalPart = strings.Split(intervalPart, ",")[0] // Remove trailing comma
				intervalPart = strings.Split(intervalPart, "#")[0] // Remove comments
				intervalPart = strings.TrimSpace(intervalPart)
				intervalPart = strings.Trim(intervalPart, `"'`) // Remove quotes
				if intervalPart != "" {
					return intervalPart, nil
				}
			}
		}

		// Also check for legacy interval variable at top level (fallback)
		if !inSettingsFunction && strings.HasPrefix(line, "interval") && strings.Contains(line, "=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				intervalPart := strings.TrimSpace(parts[1])
				intervalPart = strings.Split(intervalPart, "#")[0] // Remove comments
				intervalPart = strings.TrimSpace(intervalPart)
				intervalPart = strings.Trim(intervalPart, `"'`) // Remove quotes
				if intervalPart != "" {
					return intervalPart, nil
				}
			}
		}
	}

	// Default interval if not specified in script
	return "1m", nil
}

// Technical Indicator Functions

func (se *StrategyEngine) sma(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	result := se.indicators.calculateSMA(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) ema(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	periodInt, _ := period.Int64()
	result := se.indicators.calculateEMA(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) rsi(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	periodInt, _ := period.Int64()
	result := se.indicators.calculateRSI(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) macd(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var fast, slow, signal starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "fast?", &fast, "slow?", &slow, "signal?", &signal); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	fastInt, _ := fast.Int64()
	slowInt, _ := slow.Int64()
	signalInt, _ := signal.Int64()

	if fastInt == 0 {
		fastInt = 12
	}
	if slowInt == 0 {
		slowInt = 26
	}
	if signalInt == 0 {
		signalInt = 9
	}

	macdLine, signalLine, histogram := se.indicators.calculateMACD(priceList, int(fastInt), int(slowInt), int(signalInt))

	result := starlark.NewDict(3)
	result.SetKey(starlark.String("macd"), se.floatListToStarlark(macdLine))
	result.SetKey(starlark.String("signal"), se.floatListToStarlark(signalLine))
	result.SetKey(starlark.String("histogram"), se.floatListToStarlark(histogram))

	return result, nil
}

func (se *StrategyEngine) bollinger(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int
	var multiplier starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period?", &period, "multiplier?", &multiplier); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 20
	}

	mult := 2.0
	if multiplier != 0 {
		mult = float64(multiplier)
	}

	upper, middle, lower := se.indicators.calculateBollinger(priceList, int(periodInt), mult)

	result := starlark.NewDict(3)
	result.SetKey(starlark.String("upper"), se.floatListToStarlark(upper))
	result.SetKey(starlark.String("middle"), se.floatListToStarlark(middle))
	result.SetKey(starlark.String("lower"), se.floatListToStarlark(lower))

	return result, nil
}

func (se *StrategyEngine) stochastic(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var kPeriod, dPeriod starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "k_period?", &kPeriod, "d_period?", &dPeriod); err != nil {
		return nil, err
	}

	kPeriodInt, _ := kPeriod.Int64()
	dPeriodInt, _ := dPeriod.Int64()

	if kPeriodInt == 0 {
		kPeriodInt = 14
	}
	if dPeriodInt == 0 {
		dPeriodInt = 3
	}

	k, d := se.indicators.calculateStochastic(high.(*starlark.List), low.(*starlark.List), close.(*starlark.List), int(kPeriodInt), int(dPeriodInt))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("k"), se.floatListToStarlark(k))
	result.SetKey(starlark.String("d"), se.floatListToStarlark(d))

	return result, nil
}

// Utility functions

func (se *StrategyEngine) highest(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	periodInt, _ := period.Int64()
	result := se.indicators.calculateHighest(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) lowest(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("prices must be a list")
	}

	periodInt, _ := period.Int64()
	result := se.indicators.calculateLowest(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) crossover(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var series1, series2 starlark.Value

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "series1", &series1, "series2", &series2); err != nil {
		return nil, err
	}

	list1, ok1 := series1.(*starlark.List)
	list2, ok2 := series2.(*starlark.List)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("both series must be lists")
	}

	result := se.indicators.calculateCrossover(list1, list2)
	return se.boolListToStarlark(result), nil
}

func (se *StrategyEngine) crossunder(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var series1, series2 starlark.Value

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "series1", &series1, "series2", &series2); err != nil {
		return nil, err
	}

	list1, ok1 := series1.(*starlark.List)
	list2, ok2 := series2.(*starlark.List)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("both series must be lists")
	}

	result := se.indicators.calculateCrossunder(list1, list2)
	return se.boolListToStarlark(result), nil
}

func (se *StrategyEngine) logFunc(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg starlark.String

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "message", &msg); err != nil {
		return nil, err
	}

	se.logger.Info().Str("source", "strategy").Msg(string(msg))
	return starlark.None, nil
}

func (se *StrategyEngine) printFunc(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg starlark.String

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "message", &msg); err != nil {
		return nil, err
	}

	se.logger.Debug().Str("source", "strategy").Msg(string(msg))
	return starlark.None, nil
}

// Helper functions for data conversion

func (se *StrategyEngine) mapToStarlark(m map[string]interface{}) *starlark.Dict {
	dict := starlark.NewDict(len(m))
	for k, v := range m {
		var starlarkValue starlark.Value
		switch val := v.(type) {
		case string:
			starlarkValue = starlark.String(val)
		case int:
			starlarkValue = starlark.MakeInt(val)
		case float64:
			starlarkValue = starlark.Float(val)
		case bool:
			starlarkValue = starlark.Bool(val)
		default:
			starlarkValue = starlark.String(fmt.Sprintf("%v", val))
		}
		dict.SetKey(starlark.String(k), starlarkValue)
	}
	return dict
}

func (se *StrategyEngine) klinesToStarlark(klines []*KlineData) *starlark.List {
	list := starlark.NewList(nil)
	for _, k := range klines {
		klineDict := starlark.NewDict(6)
		klineDict.SetKey(starlark.String("timestamp"), starlark.MakeInt64(k.Timestamp.Unix()))
		klineDict.SetKey(starlark.String("open"), starlark.Float(k.Open))
		klineDict.SetKey(starlark.String("high"), starlark.Float(k.High))
		klineDict.SetKey(starlark.String("low"), starlark.Float(k.Low))
		klineDict.SetKey(starlark.String("close"), starlark.Float(k.Close))
		klineDict.SetKey(starlark.String("volume"), starlark.Float(k.Volume))
		list.Append(klineDict)
	}
	return list
}

func (se *StrategyEngine) extractPrices(klines []*KlineData, priceType string) *starlark.List {
	list := starlark.NewList(nil)
	for _, k := range klines {
		var price float64
		switch priceType {
		case "open":
			price = k.Open
		case "high":
			price = k.High
		case "low":
			price = k.Low
		case "close":
			price = k.Close
		case "volume":
			price = k.Volume
		}
		list.Append(starlark.Float(price))
	}
	return list
}

func (se *StrategyEngine) floatListToStarlark(values []float64) *starlark.List {
	list := starlark.NewList(nil)
	for _, v := range values {
		if math.IsNaN(v) {
			list.Append(starlark.None)
		} else {
			list.Append(starlark.Float(v))
		}
	}
	return list
}

func (se *StrategyEngine) boolListToStarlark(values []bool) *starlark.List {
	list := starlark.NewList(nil)
	for _, v := range values {
		list.Append(starlark.Bool(v))
	}
	return list
}
