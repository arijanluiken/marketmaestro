package rebalance

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"
	"go.starlark.net/starlark"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for rebalance actor communication
type (
	// Control messages
	StartRebalancingMsg struct{}
	StopRebalancingMsg  struct{}
	TriggerRebalanceMsg struct{}
	LoadScriptMsg       struct {
		ScriptPath string
	}
	StatusMsg struct{}

	// Actor reference messages
	SetActorReferencesMsg struct {
		ExchangePID     *actor.PID
		OrderManagerPID *actor.PID
		PortfolioPID    *actor.PID
	}

	// Data update messages
	UpdateBalancesMsg struct {
		Balances map[string]float64
	}
	UpdatePricesMsg struct {
		Prices map[string]float64
	}

	// Portfolio information
	PortfolioValueMsg struct {
		TotalValue float64
		Balances   map[string]float64
		Prices     map[string]float64
	}

	// Order placement request
	PlaceOrderMsg struct {
		Symbol    string
		Side      string
		Quantity  float64
		OrderType string
		Reason    string
	}
)

// RebalanceActor manages portfolio rebalancing using Starlark scripts
type RebalanceActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger

	// Actor references
	exchangePID     *actor.PID
	orderManagerPID *actor.PID
	portfolioPID    *actor.PID

	// Rebalancing state
	isRunning      bool
	currentScript  string
	scriptConfig   map[string]interface{}
	lastRebalance  time.Time
	rebalanceTimer *time.Timer

	// Portfolio data
	balances       map[string]float64
	prices         map[string]float64
	portfolioValue float64

	// Starlark execution
	starlarkGlobals starlark.StringDict
	scriptPath      string
	rebalanceFunc   *starlark.Function
}

// New creates a new rebalance actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *RebalanceActor {
	return &RebalanceActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
		balances:     make(map[string]float64),
		prices:       make(map[string]float64),
		scriptConfig: make(map[string]interface{}),
	}
}

// Receive handles incoming messages
func (r *RebalanceActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		r.onStarted(ctx)
	case actor.Stopped:
		r.onStopped(ctx)
	case StartRebalancingMsg:
		r.onStartRebalancing(ctx, msg)
	case StopRebalancingMsg:
		r.onStopRebalancing(ctx, msg)
	case TriggerRebalanceMsg:
		r.onTriggerRebalance(ctx, msg)
	case LoadScriptMsg:
		r.onLoadScript(ctx, msg)
	case UpdateBalancesMsg:
		r.onUpdateBalances(ctx, msg)
	case UpdatePricesMsg:
		r.onUpdatePrices(ctx, msg)
	case PortfolioValueMsg:
		r.onPortfolioValue(ctx, msg)
	case StatusMsg:
		r.onStatus(ctx)
	case SetActorReferencesMsg:
		r.onSetActorReferences(ctx, msg)
	default:
		r.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (r *RebalanceActor) onStarted(ctx *actor.Context) {
	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Rebalance actor started")

	// Initialize Starlark globals
	r.initializeStarlarkGlobals()

	// Load default rebalancing script
	defaultScript := filepath.Join("rebalance", "equal_weight.star")
	if _, err := os.Stat(defaultScript); err == nil {
		r.loadScript(defaultScript)
	}
}

func (r *RebalanceActor) onStopped(ctx *actor.Context) {
	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Rebalance actor stopped")

	if r.rebalanceTimer != nil {
		r.rebalanceTimer.Stop()
	}
}

func (r *RebalanceActor) onStartRebalancing(ctx *actor.Context, msg StartRebalancingMsg) {
	if r.isRunning {
		ctx.Respond("Already running")
		return
	}

	if r.rebalanceFunc == nil {
		ctx.Respond("No rebalancing script loaded")
		return
	}

	r.isRunning = true
	r.logger.Info().Msg("Starting automatic rebalancing")

	// Schedule periodic rebalancing based on script settings
	r.scheduleNextRebalance()
	ctx.Respond("Rebalancing started")
}

func (r *RebalanceActor) onStopRebalancing(ctx *actor.Context, msg StopRebalancingMsg) {
	r.isRunning = false
	if r.rebalanceTimer != nil {
		r.rebalanceTimer.Stop()
		r.rebalanceTimer = nil
	}

	r.logger.Info().Msg("Stopped automatic rebalancing")
	ctx.Respond("Rebalancing stopped")
}

func (r *RebalanceActor) onTriggerRebalance(ctx *actor.Context, msg TriggerRebalanceMsg) {
	r.logger.Info().Msg("Manual rebalancing triggered")
	result := r.executeRebalancing(ctx)
	ctx.Respond(result)
}

func (r *RebalanceActor) onLoadScript(ctx *actor.Context, msg LoadScriptMsg) {
	err := r.loadScript(msg.ScriptPath)
	if err != nil {
		r.logger.Error().Err(err).Str("script", msg.ScriptPath).Msg("Failed to load rebalancing script")
		ctx.Respond(fmt.Errorf("failed to load script: %w", err))
		return
	}

	r.logger.Info().Str("script", msg.ScriptPath).Msg("Rebalancing script loaded successfully")
	ctx.Respond("Script loaded successfully")
}

func (r *RebalanceActor) onUpdateBalances(ctx *actor.Context, msg UpdateBalancesMsg) {
	r.balances = msg.Balances
	r.logger.Debug().
		Int("balance_count", len(r.balances)).
		Msg("Updated portfolio balances")
}

func (r *RebalanceActor) onUpdatePrices(ctx *actor.Context, msg UpdatePricesMsg) {
	r.prices = msg.Prices
	r.logger.Debug().
		Int("price_count", len(r.prices)).
		Msg("Updated market prices")
}

func (r *RebalanceActor) onPortfolioValue(ctx *actor.Context, msg PortfolioValueMsg) {
	r.portfolioValue = msg.TotalValue
	r.balances = msg.Balances
	r.prices = msg.Prices

	r.logger.Debug().
		Float64("total_value", r.portfolioValue).
		Msg("Updated portfolio data")
}

func (r *RebalanceActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":        r.exchangeName,
		"is_running":      r.isRunning,
		"current_script":  r.currentScript,
		"last_rebalance":  r.lastRebalance,
		"portfolio_value": r.portfolioValue,
		"balance_count":   len(r.balances),
		"price_count":     len(r.prices),
		"has_script":      r.rebalanceFunc != nil,
		"timestamp":       time.Now(),
	}

	ctx.Respond(status)
}

func (r *RebalanceActor) onSetActorReferences(ctx *actor.Context, msg SetActorReferencesMsg) {
	r.exchangePID = msg.ExchangePID
	r.orderManagerPID = msg.OrderManagerPID
	r.portfolioPID = msg.PortfolioPID

	r.logger.Info().Msg("Actor references set successfully")
	ctx.Respond("OK")
}

func (r *RebalanceActor) scheduleNextRebalance() {
	if !r.isRunning {
		return
	}

	// Get rebalance interval from script config
	interval := r.getRebalanceInterval()

	r.logger.Info().
		Str("interval", interval.String()).
		Msg("Scheduling next rebalance")

	r.rebalanceTimer = time.AfterFunc(interval, func() {
		if r.isRunning {
			// Execute rebalancing (this will be called from a goroutine)
			// We need to send a message to the actor instead of calling directly
			// For now, we'll log and schedule the next one
			r.logger.Info().Msg("Rebalance timer triggered")
			r.scheduleNextRebalance()
		}
	})
}

func (r *RebalanceActor) getRebalanceInterval() time.Duration {
	if intervalStr, ok := r.scriptConfig["rebalance_interval"].(string); ok {
		if duration, err := time.ParseDuration(intervalStr); err == nil {
			return duration
		}
	}

	// Default to 1 hour
	return time.Hour
}

func (r *RebalanceActor) executeRebalancing(ctx *actor.Context) map[string]interface{} {
	if r.rebalanceFunc == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "No rebalancing function loaded",
		}
	}

	r.logger.Info().Msg("Executing rebalancing logic")

	// Update global variables for the script
	r.updateStarlarkGlobals()

	// Call the on_rebalance function
	thread := &starlark.Thread{Name: "rebalance"}
	args := starlark.Tuple{}
	kwargs := []starlark.Tuple{}

	result, err := starlark.Call(thread, r.rebalanceFunc, args, kwargs)
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to execute rebalancing function")
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	r.lastRebalance = time.Now()

	// Parse the result
	if resultDict, ok := result.(*starlark.Dict); ok {
		goResult := r.starlarkDictToGo(resultDict)
		r.logger.Info().
			Interface("result", goResult).
			Msg("Rebalancing completed")

		goResult["success"] = true
		goResult["timestamp"] = r.lastRebalance
		return goResult
	}

	return map[string]interface{}{
		"success":   true,
		"result":    result.String(),
		"timestamp": r.lastRebalance,
	}
}

func (r *RebalanceActor) loadScript(scriptPath string) error {
	r.logger.Info().Str("path", scriptPath).Msg("Loading rebalancing script")

	// Read the script file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// Parse and execute the script
	thread := &starlark.Thread{Name: "rebalance_load"}
	globals, err := starlark.ExecFile(thread, scriptPath, content, r.starlarkGlobals)
	if err != nil {
		return fmt.Errorf("failed to execute script: %w", err)
	}

	// Extract the settings function
	if settingsFunc, ok := globals["settings"]; ok {
		if fn, ok := settingsFunc.(*starlark.Function); ok {
			// Call settings() to get configuration
			args := starlark.Tuple{}
			kwargs := []starlark.Tuple{}

			result, err := starlark.Call(thread, fn, args, kwargs)
			if err != nil {
				return fmt.Errorf("failed to call settings(): %w", err)
			}

			if settingsDict, ok := result.(*starlark.Dict); ok {
				r.scriptConfig = r.starlarkDictToGo(settingsDict)
				r.logger.Info().
					Interface("config", r.scriptConfig).
					Msg("Loaded script configuration")
			}
		}
	}

	// Extract the on_rebalance function
	if rebalanceFunc, ok := globals["on_rebalance"]; ok {
		if fn, ok := rebalanceFunc.(*starlark.Function); ok {
			r.rebalanceFunc = fn
			r.currentScript = scriptPath
			r.logger.Info().Msg("Rebalancing function loaded successfully")
		} else {
			return fmt.Errorf("on_rebalance is not a function")
		}
	} else {
		return fmt.Errorf("script missing required on_rebalance function")
	}

	return nil
}

func (r *RebalanceActor) initializeStarlarkGlobals() {
	r.starlarkGlobals = starlark.StringDict{
		"config":              starlark.None,
		"get_balances":        starlark.NewBuiltin("get_balances", r.starlarkGetBalances),
		"get_current_prices":  starlark.NewBuiltin("get_current_prices", r.starlarkGetCurrentPrices),
		"get_portfolio_value": starlark.NewBuiltin("get_portfolio_value", r.starlarkGetPortfolioValue),
		"place_order":         starlark.NewBuiltin("place_order", r.starlarkPlaceOrder),
		"log":                 starlark.NewBuiltin("log", r.starlarkLog),
		"print":               starlark.NewBuiltin("print", r.starlarkPrint),
	}
}

func (r *RebalanceActor) updateStarlarkGlobals() {
	// Update config
	if len(r.scriptConfig) > 0 {
		configDict := r.goToStarlarkDict(r.scriptConfig)
		r.starlarkGlobals["config"] = configDict
	}
}

// Starlark built-in functions

func (r *RebalanceActor) starlarkGetBalances(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("get_balances() takes no arguments")
	}

	return r.goToStarlarkDict(map[string]interface{}{
		"balances": r.balances,
	}), nil
}

func (r *RebalanceActor) starlarkGetCurrentPrices(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("get_current_prices() takes no arguments")
	}

	return r.goToStarlarkDict(map[string]interface{}{
		"prices": r.prices,
	}), nil
}

func (r *RebalanceActor) starlarkGetPortfolioValue(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("get_portfolio_value() takes no arguments")
	}

	return starlark.Float(r.portfolioValue), nil
}

func (r *RebalanceActor) starlarkPlaceOrder(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var symbol, side, orderType, reason string
	var quantity float64

	// Parse keyword arguments
	for _, kw := range kwargs {
		key := string(kw[0].(starlark.String))
		value := kw[1]

		switch key {
		case "symbol":
			symbol = string(value.(starlark.String))
		case "side":
			side = string(value.(starlark.String))
		case "quantity":
			if f, ok := value.(starlark.Float); ok {
				quantity = float64(f)
			} else if i, ok := value.(starlark.Int); ok {
				if val, ok := i.Int64(); ok {
					quantity = float64(val)
				}
			}
		case "order_type":
			orderType = string(value.(starlark.String))
		case "reason":
			reason = string(value.(starlark.String))
		}
	}

	// Validate required parameters
	if symbol == "" || side == "" || quantity <= 0 {
		return r.goToStarlarkDict(map[string]interface{}{
			"success": false,
			"error":   "missing required parameters: symbol, side, quantity",
		}), nil
	}

	if orderType == "" {
		orderType = "market"
	}

	r.logger.Info().
		Str("symbol", symbol).
		Str("side", side).
		Float64("quantity", quantity).
		Str("type", orderType).
		Str("reason", reason).
		Msg("Placing rebalancing order")

	// TODO: Send order to order manager
	// For now, just return success
	return r.goToStarlarkDict(map[string]interface{}{
		"success":  true,
		"order_id": fmt.Sprintf("rebalance_%d", time.Now().Unix()),
	}), nil
}

func (r *RebalanceActor) starlarkLog(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("log() takes exactly one argument")
	}

	message := args[0].String()
	r.logger.Info().
		Str("source", "rebalance_script").
		Msg(message)

	return starlark.None, nil
}

func (r *RebalanceActor) starlarkPrint(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var messages []string
	for i := 0; i < len(args); i++ {
		messages = append(messages, args[i].String())
	}

	r.logger.Info().
		Str("source", "rebalance_script").
		Str("output", fmt.Sprintf("%v", messages)).
		Msg("Script output")

	return starlark.None, nil
}

// Helper functions for Starlark conversion

func (r *RebalanceActor) goToStarlarkDict(data map[string]interface{}) *starlark.Dict {
	dict := starlark.NewDict(len(data))
	for k, v := range data {
		key := starlark.String(k)
		var value starlark.Value

		switch val := v.(type) {
		case string:
			value = starlark.String(val)
		case int:
			value = starlark.MakeInt(val)
		case int64:
			value = starlark.MakeInt64(val)
		case float64:
			value = starlark.Float(val)
		case bool:
			value = starlark.Bool(val)
		case map[string]interface{}:
			value = r.goToStarlarkDict(val)
		case map[string]float64:
			floatDict := make(map[string]interface{})
			for fk, fv := range val {
				floatDict[fk] = fv
			}
			value = r.goToStarlarkDict(floatDict)
		default:
			value = starlark.String(fmt.Sprintf("%v", val))
		}

		dict.SetKey(key, value)
	}
	return dict
}

func (r *RebalanceActor) starlarkDictToGo(dict *starlark.Dict) map[string]interface{} {
	result := make(map[string]interface{})

	for _, item := range dict.Items() {
		key := item[0].String()
		value := item[1]

		switch val := value.(type) {
		case starlark.String:
			result[key] = string(val)
		case starlark.Int:
			if i, ok := val.Int64(); ok {
				result[key] = i
			}
		case starlark.Float:
			result[key] = float64(val)
		case starlark.Bool:
			result[key] = bool(val)
		case *starlark.Dict:
			result[key] = r.starlarkDictToGo(val)
		default:
			result[key] = value.String()
		}
	}

	return result
}
