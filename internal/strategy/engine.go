package strategy

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.starlark.net/starlark"

	"github.com/arijanluiken/mercantile/pkg/exchanges"
)

// TechnicalIndicators provides common trading indicators
type TechnicalIndicators struct {
	logger zerolog.Logger
}

// StrategyEngine executes Starlark-based trading strategies
type StrategyEngine struct {
	logger        zerolog.Logger
	indicators    *TechnicalIndicators
	builtin       starlark.StringDict
	scriptCache   map[string]*starlark.Program
	globalsCache  map[string]starlark.StringDict // Cache compiled globals for each strategy
	strategyActor interface {
		addLog(level, message string, context map[string]interface{})
	} // Interface to avoid circular import
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
		logger:       logger,
		indicators:   indicators,
		scriptCache:  make(map[string]*starlark.Program),
		globalsCache: make(map[string]starlark.StringDict),
	}

	engine.setupBuiltins()
	return engine
}

// SetStrategyActor sets the strategy actor reference for logging
func (se *StrategyEngine) SetStrategyActor(actor interface {
	addLog(level, message string, context map[string]interface{})
}) {
	se.strategyActor = actor
}

// StrategyCallbacks represents which callbacks are available in a strategy
type StrategyCallbacks struct {
	HasOnKline     bool
	HasOnOrderBook bool
	HasOnTicker    bool
	HasSettings    bool
	HasOnStart     bool
	HasOnStop      bool
}

// ValidateCallbacks checks which callbacks are available in a strategy script
func (se *StrategyEngine) ValidateCallbacks(strategyName string) (*StrategyCallbacks, error) {
	// Load strategy script
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create Starlark thread
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-validate", strategyName),
	}

	// Prepare minimal globals for script execution
	globals := starlark.StringDict{}
	for k, v := range se.builtin {
		globals[k] = v
	}
	globals["config"] = starlark.NewDict(0)

	// Execute strategy to get function definitions
	result, err := starlark.ExecFile(thread, strategyName, script, globals)
	if err != nil {
		return nil, fmt.Errorf("strategy validation failed: %w", err)
	}

	// Check which callbacks are defined
	callbacks := &StrategyCallbacks{}

	if onKlineFn, ok := result["on_kline"]; ok {
		if _, ok := onKlineFn.(*starlark.Function); ok {
			callbacks.HasOnKline = true
		}
	}

	if onOrderBookFn, ok := result["on_orderbook"]; ok {
		if _, ok := onOrderBookFn.(*starlark.Function); ok {
			callbacks.HasOnOrderBook = true
		}
	}

	if onTickerFn, ok := result["on_ticker"]; ok {
		if _, ok := onTickerFn.(*starlark.Function); ok {
			callbacks.HasOnTicker = true
		}
	}

	if settingsFn, ok := result["settings"]; ok {
		if _, ok := settingsFn.(*starlark.Function); ok {
			callbacks.HasSettings = true
		}
	}

	if onStartFn, ok := result["on_start"]; ok {
		if _, ok := onStartFn.(*starlark.Function); ok {
			callbacks.HasOnStart = true
		}
	}

	if onStopFn, ok := result["on_stop"]; ok {
		if _, ok := onStopFn.(*starlark.Function); ok {
			callbacks.HasOnStop = true
		}
	}

	se.logger.Debug().
		Str("strategy", strategyName).
		Bool("has_on_kline", callbacks.HasOnKline).
		Bool("has_on_orderbook", callbacks.HasOnOrderBook).
		Bool("has_on_ticker", callbacks.HasOnTicker).
		Bool("has_settings", callbacks.HasSettings).
		Bool("has_on_start", callbacks.HasOnStart).
		Bool("has_on_stop", callbacks.HasOnStop).
		Msg("Strategy callbacks validated")

	return callbacks, nil
}

// getOrLoadStrategy loads a strategy script and caches the compiled program and globals
func (se *StrategyEngine) getOrLoadStrategy(strategyName string) (*starlark.Program, starlark.StringDict, error) {
	// Check if already cached
	if program, exists := se.scriptCache[strategyName]; exists {
		if globals, globalsExist := se.globalsCache[strategyName]; globalsExist {
			// Return a copy of globals to avoid mutation issues
			globalsCopy := make(starlark.StringDict)
			for k, v := range globals {
				globalsCopy[k] = v
			}
			return program, globalsCopy, nil
		}
	}

	// Load the script using existing method
	script, err := se.loadStrategy(strategyName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load strategy %s: %w", strategyName, err)
	}

	// Create thread for compilation
	thread := &starlark.Thread{
		Name: fmt.Sprintf("strategy-%s-init", strategyName),
	}

	// Prepare basic globals for initial execution
	globals := make(starlark.StringDict)
	for k, v := range se.builtin {
		globals[k] = v
	}

	// Execute the script to get function definitions and initial state
	globals, err = starlark.ExecFile(thread, strategyName+".star", script, globals)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute strategy %s: %w", strategyName, err)
	}

	// Cache the initial globals
	se.globalsCache[strategyName] = globals

	se.logger.Debug().
		Str("strategy", strategyName).
		Msg("Strategy script loaded and cached")

	// Return a copy of globals to avoid mutation issues
	globalsCopy := make(starlark.StringDict)
	for k, v := range globals {
		globalsCopy[k] = v
	}

	return nil, globalsCopy, nil
}
