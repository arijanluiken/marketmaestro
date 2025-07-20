package strategy

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

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
		"round": starlark.NewBuiltin("round", func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("round() takes 1 or 2 arguments")
			}

			var num float64
			if x, ok := args[0].(starlark.Float); ok {
				num = float64(x)
			} else if x, ok := args[0].(starlark.Int); ok {
				val, _ := x.Int64()
				num = float64(val)
			} else {
				return nil, fmt.Errorf("round() requires a number")
			}

			precision := 0
			if len(args) == 2 {
				if p, ok := args[1].(starlark.Int); ok {
					precision64, _ := p.Int64()
					precision = int(precision64)
				} else {
					return nil, fmt.Errorf("round() precision must be an integer")
				}
			}

			multiplier := math.Pow(10, float64(precision))
			rounded := math.Round(num*multiplier) / multiplier

			if precision == 0 {
				return starlark.MakeInt64(int64(rounded)), nil
			}
			return starlark.Float(rounded), nil
		}),
		// Technical Indicators
		"sma":        starlark.NewBuiltin("sma", se.sma),
		"ema":        starlark.NewBuiltin("ema", se.ema),
		"rsi":        starlark.NewBuiltin("rsi", se.rsi),
		"macd":       starlark.NewBuiltin("macd", se.macd),
		"bollinger":  starlark.NewBuiltin("bollinger", se.bollinger),
		"stochastic": starlark.NewBuiltin("stochastic", se.stochastic),
		"williams_r": starlark.NewBuiltin("williams_r", se.williamsR),
		"atr":        starlark.NewBuiltin("atr", se.atr),
		"cci":        starlark.NewBuiltin("cci", se.cci),
		"vwap":       starlark.NewBuiltin("vwap", se.vwap),
		"mfi":        starlark.NewBuiltin("mfi", se.mfi),
		"stddev":     starlark.NewBuiltin("stddev", se.stddev),
		"roc":        starlark.NewBuiltin("roc", se.roc),
		// New Advanced Indicators
		"obv":           starlark.NewBuiltin("obv", se.obv),
		"adx":           starlark.NewBuiltin("adx", se.adx),
		"parabolic_sar": starlark.NewBuiltin("parabolic_sar", se.parabolicSAR),
		"keltner":       starlark.NewBuiltin("keltner", se.keltner),
		"ichimoku":      starlark.NewBuiltin("ichimoku", se.ichimoku),
		"pivot_points":  starlark.NewBuiltin("pivot_points", se.pivotPoints),
		"fibonacci":     starlark.NewBuiltin("fibonacci", se.fibonacci),
		"aroon":         starlark.NewBuiltin("aroon", se.aroon),
		// Additional Advanced Indicators
		"tsi":                 starlark.NewBuiltin("tsi", se.tsi),
		"donchian":            starlark.NewBuiltin("donchian", se.donchian),
		"advanced_cci":        starlark.NewBuiltin("advanced_cci", se.advancedCCI),
		"elder_ray":           starlark.NewBuiltin("elder_ray", se.elderRay),
		"detrended":           starlark.NewBuiltin("detrended", se.detrended),
		"kama":                starlark.NewBuiltin("kama", se.kama),
		"chaikin_oscillator":  starlark.NewBuiltin("chaikin_oscillator", se.chaikinOscillator),
		"ultimate_oscillator": starlark.NewBuiltin("ultimate_oscillator", se.ultimateOscillator),
		// Extended Advanced Indicators
		"heikin_ashi":            starlark.NewBuiltin("heikin_ashi", se.heikinAshi),
		"vortex":                 starlark.NewBuiltin("vortex", se.vortex),
		"williams_alligator":     starlark.NewBuiltin("williams_alligator", se.williamsAlligator),
		"supertrend":             starlark.NewBuiltin("supertrend", se.supertrend),
		"stochastic_rsi":         starlark.NewBuiltin("stochastic_rsi", se.stochasticRSI),
		"awesome_oscillator":     starlark.NewBuiltin("awesome_oscillator", se.awesomeOscillator),
		"accelerator_oscillator": starlark.NewBuiltin("accelerator_oscillator", se.acceleratorOscillator),
		// New Extended Indicators
		"hull_ma":              starlark.NewBuiltin("hull_ma", se.hullMA),
		"wma":                  starlark.NewBuiltin("wma", se.wma),
		"chandelier_exit":      starlark.NewBuiltin("chandelier_exit", se.chandelierExit),
		"alma":                 starlark.NewBuiltin("alma", se.alma),
		"cmo":                  starlark.NewBuiltin("cmo", se.cmo),
		"tema":                 starlark.NewBuiltin("tema", se.tema),
		"emv":                  starlark.NewBuiltin("emv", se.emv),
		"force_index":          starlark.NewBuiltin("force_index", se.forceIndex),
		"bop":                  starlark.NewBuiltin("bop", se.bop),
		"price_channel":        starlark.NewBuiltin("price_channel", se.priceChannel),
		"mass_index":           starlark.NewBuiltin("mass_index", se.massIndex),
		"volume_oscillator":    starlark.NewBuiltin("volume_oscillator", se.volumeOscillator),
		"kst":                  starlark.NewBuiltin("kst", se.kst),
		"stc":                  starlark.NewBuiltin("stc", se.stc),
		"coppock_curve":        starlark.NewBuiltin("coppock_curve", se.coppockCurve),
		"chande_kroll_stop":    starlark.NewBuiltin("chande_kroll_stop", se.chandeKrollStop),
		"elder_force_index":    starlark.NewBuiltin("elder_force_index", se.elderForceIndex),
		"klinger_oscillator":   starlark.NewBuiltin("klinger_oscillator", se.klingerOscillator),
		"volume_profile":       starlark.NewBuiltin("volume_profile", se.volumeProfile),
		// Utility functions
		"highest":    starlark.NewBuiltin("highest", se.highest),
		"lowest":     starlark.NewBuiltin("lowest", se.lowest),
		"crossover":  starlark.NewBuiltin("crossover", se.crossover),
		"crossunder": starlark.NewBuiltin("crossunder", se.crossunder),
		"log":        starlark.NewBuiltin("log", se.logFunc),
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

// ExecuteStartCallback runs the on_start callback in a strategy script
func (se *StrategyEngine) ExecuteStartCallback(strategyName string, ctx *StrategyContext) error {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-start", strategyName),
	}

	// Prepare globals with context data
	globals := se.prepareGlobals(ctx)

	// Execute strategy
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return fmt.Errorf("strategy execution failed: %w", err)
	}

	// Check if on_start function exists and call it
	if onStartFn, ok := result["on_start"]; ok {
		if fn, ok := onStartFn.(*starlark.Function); ok {
			_, err := starlark.Call(thread, fn, starlark.Tuple{}, nil)
			if err != nil {
				return fmt.Errorf("on_start callback failed: %w", err)
			}
			se.logger.Info().Str("strategy", strategyName).Msg("Strategy start callback executed successfully")
		}
	}

	return nil
}

// ExecuteStopCallback runs the on_stop callback in a strategy script
func (se *StrategyEngine) ExecuteStopCallback(strategyName string, ctx *StrategyContext) error {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-stop", strategyName),
	}

	// Prepare globals with context data
	globals := se.prepareGlobals(ctx)

	// Execute strategy
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return fmt.Errorf("strategy execution failed: %w", err)
	}

	// Check if on_stop function exists and call it
	if onStopFn, ok := result["on_stop"]; ok {
		if fn, ok := onStopFn.(*starlark.Function); ok {
			_, err := starlark.Call(thread, fn, starlark.Tuple{}, nil)
			if err != nil {
				return fmt.Errorf("on_stop callback failed: %w", err)
			}
			se.logger.Info().Str("strategy", strategyName).Msg("Strategy stop callback executed successfully")
		}
	}

	return nil
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

// New indicator functions

func (se *StrategyEngine) williamsR(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period?", &period); err != nil {
		return nil, err
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 14
	}

	result := se.indicators.calculateWilliamsR(high.(*starlark.List), low.(*starlark.List), close.(*starlark.List), int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) atr(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period?", &period); err != nil {
		return nil, err
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 14
	}

	result := se.indicators.calculateATR(high.(*starlark.List), low.(*starlark.List), close.(*starlark.List), int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) cci(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period?", &period); err != nil {
		return nil, err
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 20
	}

	result := se.indicators.calculateCCI(high.(*starlark.List), low.(*starlark.List), close.(*starlark.List), int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) vwap(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close, volume starlark.Value

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "volume", &volume); err != nil {
		return nil, err
	}

	result := se.indicators.calculateVWAP(high.(*starlark.List), low.(*starlark.List), close.(*starlark.List), volume.(*starlark.List))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) mfi(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close, volume starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "volume", &volume, "period?", &period); err != nil {
		return nil, err
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 14
	}

	result := se.indicators.calculateMFI(high.(*starlark.List), low.(*starlark.List), close.(*starlark.List), volume.(*starlark.List), int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) stddev(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	result := se.indicators.calculateStdDev(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

func (se *StrategyEngine) roc(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	result := se.indicators.calculateROC(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// obv calculates On-Balance Volume
func (se *StrategyEngine) obv(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var close, volume starlark.Value

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "close", &close, "volume", &volume); err != nil {
		return nil, err
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("close must be a list")
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("volume must be a list")
	}

	result := se.indicators.calculateOBV(closeList, volumeList)
	return se.floatListToStarlark(result), nil
}

// adx calculates Average Directional Index
func (se *StrategyEngine) adx(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period", &period); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("close must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	adxResult, plusDI, minusDI := se.indicators.calculateADX(highList, lowList, closeList, int(periodInt))

	// Return as a dict with adx, plus_di, minus_di
	result := starlark.NewDict(3)
	result.SetKey(starlark.String("adx"), se.floatListToStarlark(adxResult))
	result.SetKey(starlark.String("plus_di"), se.floatListToStarlark(plusDI))
	result.SetKey(starlark.String("minus_di"), se.floatListToStarlark(minusDI))

	return result, nil
}

// parabolicSAR calculates Parabolic Stop and Reverse
func (se *StrategyEngine) parabolicSAR(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low starlark.Value
	var step, maxStep starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "step?", &step, "max_step?", &maxStep); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("low must be a list")
	}

	stepVal := 0.02 // Default step
	if step != 0 {
		stepVal = float64(step)
	}

	maxStepVal := 0.2 // Default max step
	if maxStep != 0 {
		maxStepVal = float64(maxStep)
	}

	result := se.indicators.calculateParabolicSAR(highList, lowList, stepVal, maxStepVal)
	return se.floatListToStarlark(result), nil
}

// keltner calculates Keltner Channels
func (se *StrategyEngine) keltner(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int
	var multiplier starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period", &period, "multiplier?", &multiplier); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("close must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	multVal := 2.0 // Default multiplier
	if multiplier != 0 {
		multVal = float64(multiplier)
	}

	upper, middle, lower := se.indicators.calculateKeltnerChannels(highList, lowList, closeList, int(periodInt), multVal)

	// Return as a dict with upper, middle, lower
	result := starlark.NewDict(3)
	result.SetKey(starlark.String("upper"), se.floatListToStarlark(upper))
	result.SetKey(starlark.String("middle"), se.floatListToStarlark(middle))
	result.SetKey(starlark.String("lower"), se.floatListToStarlark(lower))

	return result, nil
}

// ichimoku calculates Ichimoku Cloud components
func (se *StrategyEngine) ichimoku(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if args.Len() < 3 {
		return nil, fmt.Errorf("ichimoku requires at least 3 arguments: high, low, close")
	}

	high := args.Index(0)
	low := args.Index(1)
	close := args.Index(2)

	// Optional parameters with defaults
	convPeriod := 9
	basePer := 26
	spanBPer := 52
	disp := 26

	// Parse optional kwargs
	for _, kw := range kwargs {
		name := string(kw[0].(starlark.String))
		switch name {
		case "conversion_period":
			if val, ok := kw[1].(starlark.Int); ok {
				v, _ := val.Int64()
				convPeriod = int(v)
			}
		case "base_period":
			if val, ok := kw[1].(starlark.Int); ok {
				v, _ := val.Int64()
				basePer = int(v)
			}
		case "span_b_period":
			if val, ok := kw[1].(starlark.Int); ok {
				v, _ := val.Int64()
				spanBPer = int(v)
			}
		case "displacement":
			if val, ok := kw[1].(starlark.Int); ok {
				v, _ := val.Int64()
				disp = int(v)
			}
		}
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("close must be a list")
	}

	tenkan, kijun, senkouA, senkouB, chikou := se.indicators.calculateIchimoku(highList, lowList, closeList, convPeriod, basePer, spanBPer, disp)

	// Return as a dict with all components
	result := starlark.NewDict(5)
	result.SetKey(starlark.String("tenkan_sen"), se.floatListToStarlark(tenkan))
	result.SetKey(starlark.String("kijun_sen"), se.floatListToStarlark(kijun))
	result.SetKey(starlark.String("senkou_span_a"), se.floatListToStarlark(senkouA))
	result.SetKey(starlark.String("senkou_span_b"), se.floatListToStarlark(senkouB))
	result.SetKey(starlark.String("chikou_span"), se.floatListToStarlark(chikou))

	return result, nil
}

// pivotPoints calculates Pivot Points and support/resistance levels
func (se *StrategyEngine) pivotPoints(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("close must be a list")
	}

	pivot, r1, r2, r3, s1, s2, s3 := se.indicators.calculatePivotPoints(highList, lowList, closeList)

	// Return as a dict with all levels
	result := starlark.NewDict(7)
	result.SetKey(starlark.String("pivot"), se.floatListToStarlark(pivot))
	result.SetKey(starlark.String("r1"), se.floatListToStarlark(r1))
	result.SetKey(starlark.String("r2"), se.floatListToStarlark(r2))
	result.SetKey(starlark.String("r3"), se.floatListToStarlark(r3))
	result.SetKey(starlark.String("s1"), se.floatListToStarlark(s1))
	result.SetKey(starlark.String("s2"), se.floatListToStarlark(s2))
	result.SetKey(starlark.String("s3"), se.floatListToStarlark(s3))

	return result, nil
}

// fibonacci calculates Fibonacci retracement levels
func (se *StrategyEngine) fibonacci(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low); err != nil {
		return nil, err
	}

	highVal := float64(high)
	lowVal := float64(low)

	if highVal <= lowVal {
		return nil, fmt.Errorf("high must be greater than low")
	}

	levels := se.indicators.calculateFibonacciRetracement(highVal, lowVal)

	// Return as a dict with all levels
	result := starlark.NewDict(len(levels))
	for level, value := range levels {
		result.SetKey(starlark.String(level), starlark.Float(value))
	}

	return result, nil
}

// aroon calculates Aroon Up and Aroon Down
func (se *StrategyEngine) aroon(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "period", &period); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("low must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}

	aroonUp, aroonDown := se.indicators.calculateAroon(highList, lowList, int(periodInt))

	// Return as a dict with aroon_up and aroon_down
	result := starlark.NewDict(2)
	result.SetKey(starlark.String("aroon_up"), se.floatListToStarlark(aroonUp))
	result.SetKey(starlark.String("aroon_down"), se.floatListToStarlark(aroonDown))

	return result, nil
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

// New technical indicator wrapper functions

// tsi calculates True Strength Index
func (se *StrategyEngine) tsi(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("tsi() takes exactly 3 arguments (prices, long_period, short_period)")
	}

	prices, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("tsi() first argument must be a list")
	}

	longPeriod, ok := args[1].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("tsi() long_period must be an integer")
	}
	longPeriodInt, _ := longPeriod.Int64()

	shortPeriod, ok := args[2].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("tsi() short_period must be an integer")
	}
	shortPeriodInt, _ := shortPeriod.Int64()

	result := se.indicators.calculateTSI(prices, int(longPeriodInt), int(shortPeriodInt))
	return se.floatListToStarlark(result), nil
}

// donchian calculates Donchian Channels
func (se *StrategyEngine) donchian(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("donchian() takes exactly 3 arguments (high, low, period)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("donchian() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("donchian() low must be a list")
	}

	period, ok := args[2].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("donchian() period must be an integer")
	}
	periodInt, _ := period.Int64()

	upper, middle, lower := se.indicators.calculateDonchianChannels(high, low, int(periodInt))

	result := starlark.NewDict(3)
	result.SetKey(starlark.String("upper"), se.floatListToStarlark(upper))
	result.SetKey(starlark.String("middle"), se.floatListToStarlark(middle))
	result.SetKey(starlark.String("lower"), se.floatListToStarlark(lower))
	return result, nil
}

// advanced_cci calculates Advanced CCI with smoothing
func (se *StrategyEngine) advancedCCI(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 5 {
		return nil, fmt.Errorf("advanced_cci() takes exactly 5 arguments (high, low, close, period, smooth_period)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("advanced_cci() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("advanced_cci() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("advanced_cci() close must be a list")
	}

	period, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("advanced_cci() period must be an integer")
	}
	periodInt, _ := period.Int64()

	smoothPeriod, ok := args[4].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("advanced_cci() smooth_period must be an integer")
	}
	smoothPeriodInt, _ := smoothPeriod.Int64()

	result := se.indicators.calculateAdvancedCCI(high, low, close, int(periodInt), int(smoothPeriodInt))
	return se.floatListToStarlark(result), nil
}

// elder_ray calculates Elder Ray Index
func (se *StrategyEngine) elderRay(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("elder_ray() takes exactly 4 arguments (high, low, close, period)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("elder_ray() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("elder_ray() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("elder_ray() close must be a list")
	}

	period, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("elder_ray() period must be an integer")
	}
	periodInt, _ := period.Int64()

	bullPower, bearPower := se.indicators.calculateElderRay(high, low, close, int(periodInt))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("bull_power"), se.floatListToStarlark(bullPower))
	result.SetKey(starlark.String("bear_power"), se.floatListToStarlark(bearPower))
	return result, nil
}

// detrended calculates Detrended Price Oscillator
func (se *StrategyEngine) detrended(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("detrended() takes exactly 2 arguments (prices, period)")
	}

	prices, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("detrended() prices must be a list")
	}

	period, ok := args[1].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("detrended() period must be an integer")
	}
	periodInt, _ := period.Int64()

	result := se.indicators.calculateDetrended(prices, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// kama calculates Kaufman Adaptive Moving Average
func (se *StrategyEngine) kama(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("kama() takes exactly 4 arguments (prices, period, fast_sc, slow_sc)")
	}

	prices, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("kama() prices must be a list")
	}

	period, ok := args[1].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("kama() period must be an integer")
	}
	periodInt, _ := period.Int64()

	fastSC, ok := args[2].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("kama() fast_sc must be an integer")
	}
	fastSCInt, _ := fastSC.Int64()

	slowSC, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("kama() slow_sc must be an integer")
	}
	slowSCInt, _ := slowSC.Int64()

	result := se.indicators.calculateKaufmanAMA(prices, int(periodInt), int(fastSCInt), int(slowSCInt))
	return se.floatListToStarlark(result), nil
}

// chaikin_oscillator calculates Chaikin Oscillator
func (se *StrategyEngine) chaikinOscillator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 6 {
		return nil, fmt.Errorf("chaikin_oscillator() takes exactly 6 arguments (high, low, close, volume, fast_period, slow_period)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chaikin_oscillator() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chaikin_oscillator() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chaikin_oscillator() close must be a list")
	}

	volume, ok := args[3].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chaikin_oscillator() volume must be a list")
	}

	fastPeriod, ok := args[4].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("chaikin_oscillator() fast_period must be an integer")
	}
	fastPeriodInt, _ := fastPeriod.Int64()

	slowPeriod, ok := args[5].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("chaikin_oscillator() slow_period must be an integer")
	}
	slowPeriodInt, _ := slowPeriod.Int64()

	result := se.indicators.calculateChaikinOscillator(high, low, close, volume, int(fastPeriodInt), int(slowPeriodInt))
	return se.floatListToStarlark(result), nil
}

// ultimate_oscillator calculates Ultimate Oscillator
func (se *StrategyEngine) ultimateOscillator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 6 {
		return nil, fmt.Errorf("ultimate_oscillator() takes exactly 6 arguments (high, low, close, period1, period2, period3)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("ultimate_oscillator() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("ultimate_oscillator() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("ultimate_oscillator() close must be a list")
	}

	period1, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("ultimate_oscillator() period1 must be an integer")
	}
	period1Int, _ := period1.Int64()

	period2, ok := args[4].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("ultimate_oscillator() period2 must be an integer")
	}
	period2Int, _ := period2.Int64()

	period3, ok := args[5].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("ultimate_oscillator() period3 must be an integer")
	}
	period3Int, _ := period3.Int64()

	result := se.indicators.calculateUltimateOscillator(high, low, close, int(period1Int), int(period2Int), int(period3Int))
	return se.floatListToStarlark(result), nil
}

// heikin_ashi calculates Heikin Ashi candlesticks
func (se *StrategyEngine) heikinAshi(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("heikin_ashi() takes exactly 4 arguments (open, high, low, close)")
	}

	open, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("heikin_ashi() open must be a list")
	}

	high, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("heikin_ashi() high must be a list")
	}

	low, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("heikin_ashi() low must be a list")
	}

	close, ok := args[3].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("heikin_ashi() close must be a list")
	}

	haOpen, haHigh, haLow, haClose := se.indicators.calculateHeikinAshi(open, high, low, close)

	result := starlark.NewDict(4)
	result.SetKey(starlark.String("open"), se.floatListToStarlark(haOpen))
	result.SetKey(starlark.String("high"), se.floatListToStarlark(haHigh))
	result.SetKey(starlark.String("low"), se.floatListToStarlark(haLow))
	result.SetKey(starlark.String("close"), se.floatListToStarlark(haClose))
	return result, nil
}

// vortex calculates Vortex Indicator
func (se *StrategyEngine) vortex(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("vortex() takes exactly 4 arguments (high, low, close, period)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("vortex() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("vortex() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("vortex() close must be a list")
	}

	period, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("vortex() period must be an integer")
	}
	periodInt, _ := period.Int64()

	viPlus, viMinus := se.indicators.calculateVortex(high, low, close, int(periodInt))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("vi_plus"), se.floatListToStarlark(viPlus))
	result.SetKey(starlark.String("vi_minus"), se.floatListToStarlark(viMinus))
	return result, nil
}

// williams_alligator calculates Williams Alligator
func (se *StrategyEngine) williamsAlligator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("williams_alligator() takes exactly 1 argument (prices)")
	}

	prices, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("williams_alligator() prices must be a list")
	}

	jaw, teeth, lips := se.indicators.calculateWilliamsAlligator(prices)

	result := starlark.NewDict(3)
	result.SetKey(starlark.String("jaw"), se.floatListToStarlark(jaw))
	result.SetKey(starlark.String("teeth"), se.floatListToStarlark(teeth))
	result.SetKey(starlark.String("lips"), se.floatListToStarlark(lips))
	return result, nil
}

// supertrend calculates Supertrend indicator
func (se *StrategyEngine) supertrend(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 5 {
		return nil, fmt.Errorf("supertrend() takes exactly 5 arguments (high, low, close, period, multiplier)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("supertrend() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("supertrend() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("supertrend() close must be a list")
	}

	period, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("supertrend() period must be an integer")
	}
	periodInt, _ := period.Int64()

	multiplier, ok := args[4].(starlark.Float)
	if !ok {
		if m, ok := args[4].(starlark.Int); ok {
			mInt, _ := m.Int64()
			multiplier = starlark.Float(mInt)
		} else {
			return nil, fmt.Errorf("supertrend() multiplier must be a number")
		}
	}

	supertrendValues, trend := se.indicators.calculateSupertrend(high, low, close, int(periodInt), float64(multiplier))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("supertrend"), se.floatListToStarlark(supertrendValues))
	result.SetKey(starlark.String("trend"), se.boolListToStarlark(trend))
	return result, nil
}

// stochastic_rsi calculates Stochastic RSI
func (se *StrategyEngine) stochasticRSI(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 5 {
		return nil, fmt.Errorf("stochastic_rsi() takes exactly 5 arguments (prices, rsi_period, stoch_period, k_period, d_period)")
	}

	prices, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("stochastic_rsi() prices must be a list")
	}

	rsiPeriod, ok := args[1].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("stochastic_rsi() rsi_period must be an integer")
	}
	rsiPeriodInt, _ := rsiPeriod.Int64()

	stochPeriod, ok := args[2].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("stochastic_rsi() stoch_period must be an integer")
	}
	stochPeriodInt, _ := stochPeriod.Int64()

	kPeriod, ok := args[3].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("stochastic_rsi() k_period must be an integer")
	}
	kPeriodInt, _ := kPeriod.Int64()

	dPeriod, ok := args[4].(starlark.Int)
	if !ok {
		return nil, fmt.Errorf("stochastic_rsi() d_period must be an integer")
	}
	dPeriodInt, _ := dPeriod.Int64()

	k, d := se.indicators.calculateStochasticRSI(prices, int(rsiPeriodInt), int(stochPeriodInt), int(kPeriodInt), int(dPeriodInt))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("k"), se.floatListToStarlark(k))
	result.SetKey(starlark.String("d"), se.floatListToStarlark(d))
	return result, nil
}

// awesome_oscillator calculates Awesome Oscillator
func (se *StrategyEngine) awesomeOscillator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("awesome_oscillator() takes exactly 2 arguments (high, low)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("awesome_oscillator() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("awesome_oscillator() low must be a list")
	}

	result := se.indicators.calculateAwesome(high, low)
	return se.floatListToStarlark(result), nil
}

// accelerator_oscillator calculates Accelerator Oscillator
func (se *StrategyEngine) acceleratorOscillator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("accelerator_oscillator() takes exactly 3 arguments (high, low, close)")
	}

	high, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("accelerator_oscillator() high must be a list")
	}

	low, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("accelerator_oscillator() low must be a list")
	}

	close, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("accelerator_oscillator() close must be a list")
	}

	result := se.indicators.calculateAcceleratorOscillator(high, low, close)
	return se.floatListToStarlark(result), nil
}

// hull_ma calculates Hull Moving Average
func (se *StrategyEngine) hullMA(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("hull_ma() prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("hull_ma() period must be positive")
	}

	result := se.indicators.calculateHullMA(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// wma calculates Weighted Moving Average
func (se *StrategyEngine) wma(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("wma() prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("wma() period must be positive")
	}

	result := se.indicators.calculateWMA(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// chandelier_exit calculates Chandelier Exit
func (se *StrategyEngine) chandelierExit(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int
	var multiplier starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period?", &period, "multiplier?", &multiplier); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chandelier_exit() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chandelier_exit() low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chandelier_exit() close must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 22
	}

	multiplierFloat := float64(multiplier)
	if multiplierFloat == 0 {
		multiplierFloat = 3.0
	}

	longExit, shortExit := se.indicators.calculateChandelierExit(highList, lowList, closeList, int(periodInt), multiplierFloat)

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("long_exit"), se.floatListToStarlark(longExit))
	result.SetKey(starlark.String("short_exit"), se.floatListToStarlark(shortExit))
	return result, nil
}

// alma calculates Arnaud Legoux Moving Average
func (se *StrategyEngine) alma(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int
	var offset, sigma starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period, "offset?", &offset, "sigma?", &sigma); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("alma() prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("alma() period must be positive")
	}

	offsetFloat := float64(offset)
	if offsetFloat == 0 {
		offsetFloat = 0.85
	}

	sigmaFloat := float64(sigma)
	if sigmaFloat == 0 {
		sigmaFloat = 6.0
	}

	result := se.indicators.calculateALMA(priceList, int(periodInt), offsetFloat, sigmaFloat)
	return se.floatListToStarlark(result), nil
}

// cmo calculates Chande Momentum Oscillator
func (se *StrategyEngine) cmo(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period?", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("cmo() prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 14
	}

	result := se.indicators.calculateCMO(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// tema calculates Triple Exponential Moving Average
func (se *StrategyEngine) tema(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "period", &period); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("tema() prices must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt <= 0 {
		return nil, fmt.Errorf("tema() period must be positive")
	}

	result := se.indicators.calculateTEMA(priceList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// emv calculates Ease of Movement
func (se *StrategyEngine) emv(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close, volume starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "volume", &volume, "period?", &period); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("emv() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("emv() low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("emv() close must be a list")
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("emv() volume must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 14
	}

	result := se.indicators.calculateEMV(highList, lowList, closeList, volumeList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// force_index calculates Force Index
func (se *StrategyEngine) forceIndex(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var close, volume starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "close", &close, "volume", &volume, "period?", &period); err != nil {
		return nil, err
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("force_index() close must be a list")
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("force_index() volume must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 13
	}

	result := se.indicators.calculateForceIndex(closeList, volumeList, int(periodInt))
	return se.floatListToStarlark(result), nil
}

// bop calculates Balance of Power
func (se *StrategyEngine) bop(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("bop() takes exactly 4 arguments (open, high, low, close)")
	}

	open, ok := args[0].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("bop() open must be a list")
	}

	high, ok := args[1].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("bop() high must be a list")
	}

	low, ok := args[2].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("bop() low must be a list")
	}

	close, ok := args[3].(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("bop() close must be a list")
	}

	result := se.indicators.calculateBOP(open, high, low, close)
	return se.floatListToStarlark(result), nil
}

// price_channel calculates Price Channel
func (se *StrategyEngine) priceChannel(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low starlark.Value
	var period starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "period?", &period); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("price_channel() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("price_channel() low must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 20
	}

	upper, middle, lower := se.indicators.calculatePriceChannel(highList, lowList, int(periodInt))

	result := starlark.NewDict(3)
	result.SetKey(starlark.String("upper"), se.floatListToStarlark(upper))
	result.SetKey(starlark.String("middle"), se.floatListToStarlark(middle))
	result.SetKey(starlark.String("lower"), se.floatListToStarlark(lower))
	return result, nil
}

// mass_index calculates Mass Index
func (se *StrategyEngine) massIndex(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low starlark.Value
	var period, sumPeriod starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "period?", &period, "sum_period?", &sumPeriod); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("mass_index() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("mass_index() low must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 9
	}

	sumPeriodInt, _ := sumPeriod.Int64()
	if sumPeriodInt == 0 {
		sumPeriodInt = 25
	}

	result := se.indicators.calculateMassIndex(highList, lowList, int(periodInt), int(sumPeriodInt))
	return se.floatListToStarlark(result), nil
}

// volume_oscillator calculates Volume Oscillator
func (se *StrategyEngine) volumeOscillator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var volume starlark.Value
	var fastPeriod, slowPeriod starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "volume", &volume, "fast_period?", &fastPeriod, "slow_period?", &slowPeriod); err != nil {
		return nil, err
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("volume_oscillator() volume must be a list")
	}

	fastPeriodInt, _ := fastPeriod.Int64()
	if fastPeriodInt == 0 {
		fastPeriodInt = 5
	}

	slowPeriodInt, _ := slowPeriod.Int64()
	if slowPeriodInt == 0 {
		slowPeriodInt = 10
	}

	result := se.indicators.calculateVolumeOscillator(volumeList, int(fastPeriodInt), int(slowPeriodInt))
	return se.floatListToStarlark(result), nil
}

// kst calculates Know Sure Thing
func (se *StrategyEngine) kst(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var roc1, roc2, roc3, roc4, sma1, sma2, sma3, sma4 starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "roc1?", &roc1, "roc2?", &roc2, "roc3?", &roc3, "roc4?", &roc4, "sma1?", &sma1, "sma2?", &sma2, "sma3?", &sma3, "sma4?", &sma4); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("kst() prices must be a list")
	}

	// Default KST parameters
	roc1Int, _ := roc1.Int64()
	if roc1Int == 0 {
		roc1Int = 10
	}
	roc2Int, _ := roc2.Int64()
	if roc2Int == 0 {
		roc2Int = 15
	}
	roc3Int, _ := roc3.Int64()
	if roc3Int == 0 {
		roc3Int = 20
	}
	roc4Int, _ := roc4.Int64()
	if roc4Int == 0 {
		roc4Int = 30
	}

	sma1Int, _ := sma1.Int64()
	if sma1Int == 0 {
		sma1Int = 10
	}
	sma2Int, _ := sma2.Int64()
	if sma2Int == 0 {
		sma2Int = 10
	}
	sma3Int, _ := sma3.Int64()
	if sma3Int == 0 {
		sma3Int = 10
	}
	sma4Int, _ := sma4.Int64()
	if sma4Int == 0 {
		sma4Int = 15
	}

	kstLine, signal := se.indicators.calculateKST(priceList, int(roc1Int), int(roc2Int), int(roc3Int), int(roc4Int), int(sma1Int), int(sma2Int), int(sma3Int), int(sma4Int))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("kst"), se.floatListToStarlark(kstLine))
	result.SetKey(starlark.String("signal"), se.floatListToStarlark(signal))
	return result, nil
}

// stc calculates Schaff Trend Cycle
func (se *StrategyEngine) stc(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var fastPeriod, slowPeriod, cyclePeriod starlark.Int
	var factor starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "fast_period?", &fastPeriod, "slow_period?", &slowPeriod, "cycle_period?", &cyclePeriod, "factor?", &factor); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("stc() prices must be a list")
	}

	fastPeriodInt, _ := fastPeriod.Int64()
	if fastPeriodInt == 0 {
		fastPeriodInt = 23
	}

	slowPeriodInt, _ := slowPeriod.Int64()
	if slowPeriodInt == 0 {
		slowPeriodInt = 50
	}

	cyclePeriodInt, _ := cyclePeriod.Int64()
	if cyclePeriodInt == 0 {
		cyclePeriodInt = 10
	}

	factorFloat := float64(factor)
	if factorFloat == 0 {
		factorFloat = 0.5
	}

	result := se.indicators.calculateSTC(priceList, int(fastPeriodInt), int(slowPeriodInt), int(cyclePeriodInt), factorFloat)
	return se.floatListToStarlark(result), nil
}

// coppock_curve calculates Coppock Curve
func (se *StrategyEngine) coppockCurve(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prices starlark.Value
	var roc1, roc2, wma starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "prices", &prices, "roc1_period?", &roc1, "roc2_period?", &roc2, "wma_period?", &wma); err != nil {
		return nil, err
	}

	priceList, ok := prices.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("coppock_curve() prices must be a list")
	}

	roc1Int, _ := roc1.Int64()
	if roc1Int == 0 {
		roc1Int = 14
	}

	roc2Int, _ := roc2.Int64()
	if roc2Int == 0 {
		roc2Int = 11
	}

	wmaInt, _ := wma.Int64()
	if wmaInt == 0 {
		wmaInt = 10
	}

	result := se.indicators.calculateCoppockCurve(priceList, int(roc1Int), int(roc2Int), int(wmaInt))
	return se.floatListToStarlark(result), nil
}

// chande_kroll_stop calculates Chande Kroll Stop
func (se *StrategyEngine) chandeKrollStop(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close starlark.Value
	var period starlark.Int
	var multiplier starlark.Float

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "period?", &period, "multiplier?", &multiplier); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chande_kroll_stop() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chande_kroll_stop() low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("chande_kroll_stop() close must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 10
	}

	multiplierFloat := float64(multiplier)
	if multiplierFloat == 0 {
		multiplierFloat = 3.0
	}

	longStop, shortStop := se.indicators.calculateChandeKrollStop(highList, lowList, closeList, int(periodInt), multiplierFloat)

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("long_stop"), se.floatListToStarlark(longStop))
	result.SetKey(starlark.String("short_stop"), se.floatListToStarlark(shortStop))
	return result, nil
}

// elder_force_index calculates Elder's Force Index
func (se *StrategyEngine) elderForceIndex(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var close, volume starlark.Value
	var shortPeriod, longPeriod starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "close", &close, "volume", &volume, "short_period?", &shortPeriod, "long_period?", &longPeriod); err != nil {
		return nil, err
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("elder_force_index() close must be a list")
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("elder_force_index() volume must be a list")
	}

	shortPeriodInt, _ := shortPeriod.Int64()
	if shortPeriodInt == 0 {
		shortPeriodInt = 2
	}

	longPeriodInt, _ := longPeriod.Int64()
	if longPeriodInt == 0 {
		longPeriodInt = 13
	}

	shortFI, longFI := se.indicators.calculateElderForceIndex(closeList, volumeList, int(shortPeriodInt), int(longPeriodInt))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("short"), se.floatListToStarlark(shortFI))
	result.SetKey(starlark.String("long"), se.floatListToStarlark(longFI))
	return result, nil
}

// klinger_oscillator calculates Klinger Volume Oscillator
func (se *StrategyEngine) klingerOscillator(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close, volume starlark.Value
	var fastPeriod, slowPeriod, signalPeriod starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "volume", &volume, "fast_period?", &fastPeriod, "slow_period?", &slowPeriod, "signal_period?", &signalPeriod); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("klinger_oscillator() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("klinger_oscillator() low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("klinger_oscillator() close must be a list")
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("klinger_oscillator() volume must be a list")
	}

	fastPeriodInt, _ := fastPeriod.Int64()
	if fastPeriodInt == 0 {
		fastPeriodInt = 34
	}

	slowPeriodInt, _ := slowPeriod.Int64()
	if slowPeriodInt == 0 {
		slowPeriodInt = 55
	}

	signalPeriodInt, _ := signalPeriod.Int64()
	if signalPeriodInt == 0 {
		signalPeriodInt = 13
	}

	ko, signal := se.indicators.calculateKlingerOscillator(highList, lowList, closeList, volumeList, int(fastPeriodInt), int(slowPeriodInt), int(signalPeriodInt))

	result := starlark.NewDict(2)
	result.SetKey(starlark.String("oscillator"), se.floatListToStarlark(ko))
	result.SetKey(starlark.String("signal"), se.floatListToStarlark(signal))
	return result, nil
}

// volume_profile calculates Volume Profile
func (se *StrategyEngine) volumeProfile(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var high, low, close, volume starlark.Value
	var period, levels starlark.Int

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "high", &high, "low", &low, "close", &close, "volume", &volume, "period?", &period, "levels?", &levels); err != nil {
		return nil, err
	}

	highList, ok := high.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("volume_profile() high must be a list")
	}

	lowList, ok := low.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("volume_profile() low must be a list")
	}

	closeList, ok := close.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("volume_profile() close must be a list")
	}

	volumeList, ok := volume.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("volume_profile() volume must be a list")
	}

	periodInt, _ := period.Int64()
	if periodInt == 0 {
		periodInt = 100
	}

	levelsInt, _ := levels.Int64()
	if levelsInt == 0 {
		levelsInt = 20
	}

	profile := se.indicators.calculateVolumeProfile(highList, lowList, closeList, volumeList, int(periodInt), int(levelsInt))

	// Convert map to Starlark dict
	profileDict := starlark.NewDict(len(profile))
	for price, vol := range profile {
		profileDict.SetKey(starlark.Float(price), starlark.Float(vol))
	}

	return profileDict, nil
}
