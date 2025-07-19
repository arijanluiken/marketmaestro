package risk

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/internal/settings"
	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for risk management
type (
	// Risk check messages
	ValidateOrderMsg struct {
		Exchange string
		Symbol   string
		Side     string
		Quantity float64
		Price    float64
	}

	// Risk check response
	OrderValidationResponse struct {
		Approved bool     `json:"approved"`
		Reason   string   `json:"reason,omitempty"`
		Warnings []string `json:"warnings,omitempty"`
	}

	// Portfolio update for risk calculations
	UpdatePortfolioValueMsg struct {
		TotalValue float64
		Cash       float64
	}

	// Risk metrics query
	GetRiskMetricsMsg struct{}

	// Risk metrics response
	RiskMetricsResponse struct {
		MaxDrawdown           float64            `json:"max_drawdown"`
		VaR95                 float64            `json:"var_95"`
		PositionConcentration map[string]float64 `json:"position_concentration"`
		LeverageRatio         float64            `json:"leverage_ratio"`
		DailyRiskLimit        float64            `json:"daily_risk_limit"`
		DailyRiskUsed         float64            `json:"daily_risk_used"`
	}

	// Risk configuration messages
	SetRiskParameterMsg struct {
		Key   string
		Value string
	}

	GetRiskParameterMsg struct {
		Key string
	}

	LoadRiskConfigMsg struct{}

	// Risk parameter response
	RiskParameterResponse struct {
		Parameter string      `json:"parameter"`
		Value     interface{} `json:"value"`
		Found     bool        `json:"found"`
	}

	// Status message
	StatusMsg struct{}

	// Actor reference messages
	SetSettingsActorMsg struct {
		SettingsPID *actor.PID
	}
)

// Risk configuration parameters
type RiskConfig struct {
	MaxPositionSize    float64 // Max position size as percentage of portfolio
	MaxDailyLoss       float64 // Max daily loss in base currency
	MaxDailyVolume     float64 // Max daily volume as multiple of portfolio value
	MaxDailyRisk       float64 // Max daily risk as percentage of portfolio
	MaxDrawdown        float64 // Max drawdown from high water mark
	MaxOpenPositions   int     // Max number of open positions
	MaxPortfolioRisk   float64 // Max portfolio risk
	MaxCorrelation     float64 // Max correlation between positions
	MaxLeverage        float64 // Max leverage allowed
	MaxDailyTrades     int     // Max daily trades
	MaxHourlyTrades    int     // Max hourly trades
	VaRLimit           float64 // Value at Risk limit
	MaxDrawdownLimit   float64 // Max drawdown limit
	ConcentrationLimit float64 // Position concentration limit
}

// Default risk configuration
func defaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		MaxPositionSize:    0.1,    // 10% of portfolio per position
		MaxDailyLoss:       1000.0, // $1000 max daily loss
		MaxDailyVolume:     2.0,    // 200% of portfolio value traded per day
		MaxDailyRisk:       0.2,    // 20% of portfolio at risk per day
		MaxDrawdown:        0.15,   // 15% drawdown from high water mark
		MaxOpenPositions:   5,      // Max 5 open positions
		MaxPortfolioRisk:   0.25,   // 25% of portfolio at risk
		MaxCorrelation:     0.7,    // 70% max correlation
		MaxLeverage:        3.0,    // 3x max leverage
		MaxDailyTrades:     20,     // 20 trades per day
		MaxHourlyTrades:    5,      // 5 trades per hour
		VaRLimit:           0.05,   // 5% VaR limit
		MaxDrawdownLimit:   0.20,   // 20% max drawdown
		ConcentrationLimit: 0.30,   // 30% max concentration
	}
}

// Risk tracking data structures
type OrderHistory struct {
	Timestamp time.Time
	Exchange  string
	Symbol    string
	Side      string
	Quantity  float64
	Price     float64
	Value     float64
}

type PositionRisk struct {
	Symbol        string
	Value         float64
	Concentration float64
	VaR           float64
}

// RiskManagerActor manages risk controls and validation
type RiskManagerActor struct {
	exchangeName   string
	config         *config.Config
	db             *database.DB
	logger         zerolog.Logger
	portfolioValue float64
	cash           float64
	orderHistory   []OrderHistory
	dailyVolume    map[string]float64 // date -> volume
	maxDrawdown    float64
	highWaterMark  float64
	dailyRiskUsed  float64

	// Actor references
	settingsPID *actor.PID
	riskConfig  *RiskConfig
}

// New creates a new risk management actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *RiskManagerActor {
	return &RiskManagerActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
		orderHistory: make([]OrderHistory, 0),
		dailyVolume:  make(map[string]float64),
		riskConfig:   defaultRiskConfig(),
	}
}

// SetSettingsActor sets the reference to the settings actor
func (r *RiskManagerActor) SetSettingsActor(settingsPID *actor.PID) {
	r.settingsPID = settingsPID
}

// Receive handles incoming messages
func (r *RiskManagerActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		r.onStarted(ctx)
	case actor.Stopped:
		r.onStopped(ctx)
	case ValidateOrderMsg:
		r.onValidateOrder(ctx, msg)
	case UpdatePortfolioValueMsg:
		r.onUpdatePortfolioValue(ctx, msg)
	case GetRiskMetricsMsg:
		r.onGetRiskMetrics(ctx)
	case SetRiskParameterMsg:
		r.onSetRiskParameter(ctx, msg)
	case GetRiskParameterMsg:
		r.onGetRiskParameter(ctx, msg)
	case LoadRiskConfigMsg:
		r.onLoadRiskConfig(ctx, msg)
	case SetSettingsActorMsg:
		r.onSetSettingsActor(ctx, msg)
	case StatusMsg:
		r.onStatus(ctx)
	default:
		r.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (r *RiskManagerActor) onStarted(ctx *actor.Context) {
	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Risk manager actor started")

	// Initialize with default values
	r.portfolioValue = 100000.0 // Default starting portfolio
	r.cash = 50000.0
	r.highWaterMark = r.portfolioValue

	// Load risk configuration from settings if available
	if r.settingsPID != nil {
		r.loadRiskConfigFromSettings(ctx)
	}

	// Start periodic risk calculations
	r.schedulePeriodicTasks(ctx)
}

func (r *RiskManagerActor) onStopped(ctx *actor.Context) {
	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Risk manager actor stopped")
}

func (r *RiskManagerActor) onValidateOrder(ctx *actor.Context, msg ValidateOrderMsg) {
	response := r.validateOrder(msg)

	if response.Approved {
		// Record the order in history
		orderValue := msg.Quantity * msg.Price
		r.orderHistory = append(r.orderHistory, OrderHistory{
			Timestamp: time.Now(),
			Exchange:  msg.Exchange,
			Symbol:    msg.Symbol,
			Side:      msg.Side,
			Quantity:  msg.Quantity,
			Price:     msg.Price,
			Value:     orderValue,
		})

		// Update daily risk usage
		r.dailyRiskUsed += orderValue

		// Update daily volume
		today := time.Now().Format("2006-01-02")
		r.dailyVolume[today] += orderValue

		r.logger.Info().
			Str("exchange", r.exchangeName).
			Str("symbol", msg.Symbol).
			Str("side", msg.Side).
			Float64("quantity", msg.Quantity).
			Float64("price", msg.Price).
			Float64("value", orderValue).
			Msg("Order approved by risk management")
	} else {
		r.logger.Warn().
			Str("exchange", r.exchangeName).
			Str("symbol", msg.Symbol).
			Str("side", msg.Side).
			Float64("quantity", msg.Quantity).
			Float64("price", msg.Price).
			Str("reason", response.Reason).
			Msg("Order rejected by risk management")
	}

	ctx.Respond(response)
}

func (r *RiskManagerActor) validateOrder(msg ValidateOrderMsg) OrderValidationResponse {
	var warnings []string
	orderValue := msg.Quantity * msg.Price

	// Check 1: Position size limit
	maxPositionValue := r.portfolioValue * r.config.Risk.MaxPositionSize
	if orderValue > maxPositionValue {
		return OrderValidationResponse{
			Approved: false,
			Reason:   fmt.Sprintf("Order value %.2f exceeds max position size limit %.2f", orderValue, maxPositionValue),
		}
	}

	// Check 2: Daily volume limit
	today := time.Now().Format("2006-01-02")
	todayVolume := r.dailyVolume[today]
	maxDailyVolume := r.portfolioValue * r.config.Risk.MaxDailyVolume

	if todayVolume+orderValue > maxDailyVolume {
		return OrderValidationResponse{
			Approved: false,
			Reason:   fmt.Sprintf("Order would exceed daily volume limit. Current: %.2f, Limit: %.2f", todayVolume+orderValue, maxDailyVolume),
		}
	}

	// Check 3: Available cash for buy orders
	if msg.Side == "buy" && orderValue > r.cash {
		return OrderValidationResponse{
			Approved: false,
			Reason:   fmt.Sprintf("Insufficient cash. Required: %.2f, Available: %.2f", orderValue, r.cash),
		}
	}

	// Check 4: Daily risk limit
	maxDailyRisk := r.portfolioValue * r.config.Risk.MaxDailyRisk
	if r.dailyRiskUsed+orderValue > maxDailyRisk {
		return OrderValidationResponse{
			Approved: false,
			Reason:   fmt.Sprintf("Order would exceed daily risk limit. Current: %.2f, Limit: %.2f", r.dailyRiskUsed+orderValue, maxDailyRisk),
		}
	}

	// Check 5: Drawdown limit
	currentDrawdown := (r.highWaterMark - r.portfolioValue) / r.highWaterMark
	if currentDrawdown > r.config.Risk.MaxDrawdown {
		return OrderValidationResponse{
			Approved: false,
			Reason:   fmt.Sprintf("Current drawdown %.2f%% exceeds maximum allowed %.2f%%", currentDrawdown*100, r.config.Risk.MaxDrawdown*100),
		}
	}

	// Warning checks
	if orderValue > maxPositionValue*0.8 {
		warnings = append(warnings, "Order size is close to position limit")
	}

	if todayVolume+orderValue > maxDailyVolume*0.8 {
		warnings = append(warnings, "Approaching daily volume limit")
	}

	if currentDrawdown > r.config.Risk.MaxDrawdown*0.8 {
		warnings = append(warnings, "Approaching maximum drawdown limit")
	}

	return OrderValidationResponse{
		Approved: true,
		Warnings: warnings,
	}
}

func (r *RiskManagerActor) onUpdatePortfolioValue(ctx *actor.Context, msg UpdatePortfolioValueMsg) {
	r.portfolioValue = msg.TotalValue
	r.cash = msg.Cash

	// Update high water mark and drawdown
	if r.portfolioValue > r.highWaterMark {
		r.highWaterMark = r.portfolioValue
	}

	currentDrawdown := (r.highWaterMark - r.portfolioValue) / r.highWaterMark
	if currentDrawdown > r.maxDrawdown {
		r.maxDrawdown = currentDrawdown
	}

	r.logger.Debug().
		Str("exchange", r.exchangeName).
		Float64("portfolio_value", r.portfolioValue).
		Float64("cash", r.cash).
		Float64("high_water_mark", r.highWaterMark).
		Float64("current_drawdown", currentDrawdown).
		Float64("max_drawdown", r.maxDrawdown).
		Msg("Portfolio value updated")
}

func (r *RiskManagerActor) onGetRiskMetrics(ctx *actor.Context) {
	positionConcentration := r.calculatePositionConcentration()
	leverageRatio := r.calculateLeverageRatio()
	var95 := r.calculateVaR95()

	response := RiskMetricsResponse{
		MaxDrawdown:           r.maxDrawdown,
		VaR95:                 var95,
		PositionConcentration: positionConcentration,
		LeverageRatio:         leverageRatio,
		DailyRiskLimit:        r.portfolioValue * r.config.Risk.MaxDailyRisk,
		DailyRiskUsed:         r.dailyRiskUsed,
	}

	ctx.Respond(response)
}

func (r *RiskManagerActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":        r.exchangeName,
		"timestamp":       time.Now(),
		"portfolio_value": r.portfolioValue,
		"cash":            r.cash,
		"max_drawdown":    r.maxDrawdown,
		"daily_risk_used": r.dailyRiskUsed,
		"orders_today":    r.getOrdersToday(),
	}

	ctx.Respond(status)
}

func (r *RiskManagerActor) calculatePositionConcentration() map[string]float64 {
	concentration := make(map[string]float64)

	// Sample calculation - in real implementation, this would use actual positions
	concentration["BTCUSDT"] = 0.25 // 25% of portfolio in BTC
	concentration["ETHUSDT"] = 0.15 // 15% of portfolio in ETH
	concentration["CASH"] = 0.60    // 60% in cash

	return concentration
}

func (r *RiskManagerActor) calculateLeverageRatio() float64 {
	// Sample calculation - in real implementation, this would use actual margin positions
	return 1.0 // No leverage
}

func (r *RiskManagerActor) calculateVaR95() float64 {
	// Simplified VaR calculation
	// In real implementation, this would use historical returns and proper statistics
	if len(r.orderHistory) < 30 {
		return r.portfolioValue * 0.05 // 5% of portfolio
	}

	// Calculate portfolio volatility from order history
	var returns []float64
	for i := 1; i < len(r.orderHistory) && i < 30; i++ {
		if r.orderHistory[i-1].Value > 0 {
			ret := (r.orderHistory[i].Value - r.orderHistory[i-1].Value) / r.orderHistory[i-1].Value
			returns = append(returns, ret)
		}
	}

	if len(returns) == 0 {
		return r.portfolioValue * 0.05
	}

	// Calculate standard deviation
	var sum, mean float64
	for _, ret := range returns {
		sum += ret
	}
	mean = sum / float64(len(returns))

	var variance float64
	for _, ret := range returns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	// 95% VaR (1.645 standard deviations)
	return r.portfolioValue * stdDev * 1.645
}

func (r *RiskManagerActor) getOrdersToday() int {
	today := time.Now().Format("2006-01-02")
	count := 0

	for _, order := range r.orderHistory {
		if order.Timestamp.Format("2006-01-02") == today {
			count++
		}
	}

	return count
}

func (r *RiskManagerActor) schedulePeriodicTasks(ctx *actor.Context) {
	// Reset daily counters at midnight UTC
	go func() {
		// Calculate time until next midnight UTC
		now := time.Now().UTC()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		time.Sleep(nextMidnight.Sub(now))

		// Reset daily counters immediately
		r.resetDailyCounters()

		// Now start the 24-hour ticker for subsequent resets
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			r.resetDailyCounters()
		}
	}()

	// Update risk metrics every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			r.updateRiskMetrics()
		}
	}()
}

func (r *RiskManagerActor) resetDailyCounters() {
	r.dailyRiskUsed = 0

	// Clean up old daily volume data (keep only last 30 days)
	cutoff := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	for date := range r.dailyVolume {
		if date < cutoff {
			delete(r.dailyVolume, date)
		}
	}

	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Daily risk counters reset")
}

func (r *RiskManagerActor) updateRiskMetrics() {
	// Clean up old order history (keep only last 1000 orders)
	if len(r.orderHistory) > 1000 {
		r.orderHistory = r.orderHistory[len(r.orderHistory)-1000:]
	}

	r.logger.Debug().
		Str("exchange", r.exchangeName).
		Int("order_history_size", len(r.orderHistory)).
		Float64("max_drawdown", r.maxDrawdown).
		Float64("daily_risk_used", r.dailyRiskUsed).
		Msg("Risk metrics updated")
}

// onSetRiskParameter handles setting a risk parameter
func (r *RiskManagerActor) onSetRiskParameter(ctx *actor.Context, msg SetRiskParameterMsg) {
	if r.settingsPID == nil {
		r.logger.Error().Msg("Settings actor not configured")
		ctx.Respond(fmt.Errorf("settings actor not configured"))
		return
	}

	// Store in settings actor
	settingKey := fmt.Sprintf("risk.%s", msg.Key)
	settingMsg := settings.SetSettingMsg{
		Key:   settingKey,
		Value: msg.Value,
	}

	result, err := ctx.Request(r.settingsPID, settingMsg, 5*time.Second).Result()
	if err != nil {
		r.logger.Error().Err(err).Str("parameter", msg.Key).Msg("Failed to store risk parameter")
		ctx.Respond(fmt.Errorf("failed to store risk parameter: %w", err))
		return
	}

	if errorResult, ok := result.(error); ok {
		r.logger.Error().Err(errorResult).Str("parameter", msg.Key).Msg("Settings actor returned error")
		ctx.Respond(errorResult)
		return
	}

	// Update local config
	r.updateLocalRiskConfig(msg.Key, msg.Value)

	r.logger.Info().
		Str("parameter", msg.Key).
		Interface("value", msg.Value).
		Msg("Risk parameter updated")

	ctx.Respond("OK")
}

// onGetRiskParameter handles getting a risk parameter
func (r *RiskManagerActor) onGetRiskParameter(ctx *actor.Context, msg GetRiskParameterMsg) {
	if r.settingsPID == nil {
		r.logger.Error().Msg("Settings actor not configured")
		ctx.Respond(RiskParameterResponse{
			Parameter: msg.Key,
			Value:     r.getLocalRiskParameter(msg.Key),
			Found:     true, // Use local config as fallback
		})
		return
	}

	// Get from settings actor
	settingKey := fmt.Sprintf("risk.%s", msg.Key)
	settingMsg := settings.GetSettingMsg{Key: settingKey}

	result, err := ctx.Request(r.settingsPID, settingMsg, 5*time.Second).Result()
	if err != nil {
		r.logger.Error().Err(err).Str("parameter", msg.Key).Msg("Failed to get risk parameter")
		ctx.Respond(RiskParameterResponse{
			Parameter: msg.Key,
			Value:     r.getLocalRiskParameter(msg.Key),
			Found:     true, // Use local config as fallback
		})
		return
	}

	if settingResp, ok := result.(settings.SettingResponse); ok {
		if settingResp.Found {
			ctx.Respond(RiskParameterResponse{
				Parameter: msg.Key,
				Value:     settingResp.Value,
				Found:     true,
			})
		} else {
			// Return local config value
			ctx.Respond(RiskParameterResponse{
				Parameter: msg.Key,
				Value:     r.getLocalRiskParameter(msg.Key),
				Found:     true,
			})
		}
		return
	}

	r.logger.Error().Str("parameter", msg.Key).Msg("Unexpected response from settings actor")
	ctx.Respond(RiskParameterResponse{
		Parameter: msg.Key,
		Value:     r.getLocalRiskParameter(msg.Key),
		Found:     true,
	})
}

// onLoadRiskConfig handles loading the complete risk configuration
func (r *RiskManagerActor) onLoadRiskConfig(ctx *actor.Context, msg LoadRiskConfigMsg) {
	if r.settingsPID == nil {
		r.logger.Error().Msg("Settings actor not configured")
		ctx.Respond(r.riskConfig) // Return current config
		return
	}

	r.loadRiskConfigFromSettings(ctx)
	ctx.Respond(r.riskConfig)
}

// loadRiskConfigFromSettings loads risk configuration from the settings actor
func (r *RiskManagerActor) loadRiskConfigFromSettings(ctx *actor.Context) {
	if r.settingsPID == nil {
		r.logger.Warn().Msg("Settings actor not configured, using default risk config")
		return
	}

	r.logger.Info().Msg("Loading risk configuration from settings")

	// Load each risk parameter
	parameters := []string{
		"max_position_size",
		"max_daily_loss",
		"max_portfolio_risk",
		"max_correlation",
		"max_leverage",
		"max_daily_trades",
		"max_hourly_trades",
		"var_limit",
		"max_drawdown_limit",
		"concentration_limit",
	}

	for _, param := range parameters {
		settingKey := fmt.Sprintf("risk.%s", param)
		settingMsg := settings.GetSettingMsg{Key: settingKey}

		result, err := ctx.Request(r.settingsPID, settingMsg, 5*time.Second).Result()
		if err != nil {
			r.logger.Error().Err(err).Str("parameter", param).Msg("Failed to load risk parameter")
			continue
		}

		if settingResp, ok := result.(settings.SettingResponse); ok && settingResp.Found {
			r.updateLocalRiskConfigFromString(param, settingResp.Value)
		}
	}

	r.logger.Info().Msg("Risk configuration loaded from settings")
}

// onSetSettingsActor handles setting the settings actor reference
func (r *RiskManagerActor) onSetSettingsActor(ctx *actor.Context, msg SetSettingsActorMsg) {
	r.settingsPID = msg.SettingsPID
	r.logger.Info().Msg("Settings actor reference set, loading risk configuration")

	// Load risk configuration from settings
	r.loadRiskConfigFromSettings(ctx)
}

// updateLocalRiskConfig updates the local risk configuration
func (r *RiskManagerActor) updateLocalRiskConfig(parameter string, value interface{}) {
	switch parameter {
	case "max_position_size":
		if v, ok := value.(float64); ok {
			r.riskConfig.MaxPositionSize = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.MaxPositionSize = v
			}
		}
	case "max_daily_loss":
		if v, ok := value.(float64); ok {
			r.riskConfig.MaxDailyLoss = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.MaxDailyLoss = v
			}
		}
	case "max_portfolio_risk":
		if v, ok := value.(float64); ok {
			r.riskConfig.MaxPortfolioRisk = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.MaxPortfolioRisk = v
			}
		}
	case "max_correlation":
		if v, ok := value.(float64); ok {
			r.riskConfig.MaxCorrelation = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.MaxCorrelation = v
			}
		}
	case "max_leverage":
		if v, ok := value.(float64); ok {
			r.riskConfig.MaxLeverage = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.MaxLeverage = v
			}
		}
	case "max_daily_trades":
		if v, ok := value.(int); ok {
			r.riskConfig.MaxDailyTrades = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.Atoi(s); err == nil {
				r.riskConfig.MaxDailyTrades = v
			}
		}
	case "max_hourly_trades":
		if v, ok := value.(int); ok {
			r.riskConfig.MaxHourlyTrades = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.Atoi(s); err == nil {
				r.riskConfig.MaxHourlyTrades = v
			}
		}
	case "var_limit":
		if v, ok := value.(float64); ok {
			r.riskConfig.VaRLimit = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.VaRLimit = v
			}
		}
	case "max_drawdown_limit":
		if v, ok := value.(float64); ok {
			r.riskConfig.MaxDrawdownLimit = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.MaxDrawdownLimit = v
			}
		}
	case "concentration_limit":
		if v, ok := value.(float64); ok {
			r.riskConfig.ConcentrationLimit = v
		} else if s, ok := value.(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				r.riskConfig.ConcentrationLimit = v
			}
		}
	}
}

// updateLocalRiskConfigFromString updates the local risk configuration from string values
func (r *RiskManagerActor) updateLocalRiskConfigFromString(parameter, value string) {
	switch parameter {
	case "max_position_size", "max_daily_loss", "max_portfolio_risk",
		"max_correlation", "max_leverage", "var_limit",
		"max_drawdown_limit", "concentration_limit":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			r.updateLocalRiskConfig(parameter, v)
		}
	case "max_daily_trades", "max_hourly_trades":
		if v, err := strconv.Atoi(value); err == nil {
			r.updateLocalRiskConfig(parameter, v)
		}
	}
}

// getLocalRiskParameter gets a risk parameter from local configuration
func (r *RiskManagerActor) getLocalRiskParameter(parameter string) interface{} {
	switch parameter {
	case "max_position_size":
		return r.riskConfig.MaxPositionSize
	case "max_daily_loss":
		return r.riskConfig.MaxDailyLoss
	case "max_portfolio_risk":
		return r.riskConfig.MaxPortfolioRisk
	case "max_correlation":
		return r.riskConfig.MaxCorrelation
	case "max_leverage":
		return r.riskConfig.MaxLeverage
	case "max_daily_trades":
		return r.riskConfig.MaxDailyTrades
	case "max_hourly_trades":
		return r.riskConfig.MaxHourlyTrades
	case "var_limit":
		return r.riskConfig.VaRLimit
	case "max_drawdown_limit":
		return r.riskConfig.MaxDrawdownLimit
	case "concentration_limit":
		return r.riskConfig.ConcentrationLimit
	default:
		return nil
	}
}
