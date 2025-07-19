package risk

import (
	"fmt"
	"math"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

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

	// Status message
	StatusMsg struct{}
)

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
	}
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
	// Reset daily counters at midnight
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.resetDailyCounters()
			}
		}
	}()

	// Update risk metrics every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.updateRiskMetrics()
			}
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
