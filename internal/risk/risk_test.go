package risk

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

func setupTestDatabase(t *testing.T) *database.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return db
}

func setupTestRiskManager(t *testing.T) (*RiskManagerActor, *database.DB) {
	db := setupTestDatabase(t)
	cfg := &config.Config{
		Risk: config.RiskConfig{
			MaxPositionSize:  0.1,
			MaxDailyLoss:     1000.0,
			MaxDailyVolume:   2.0,
			MaxDailyRisk:     0.2,
			MaxDrawdown:      0.15,
			MaxOpenPositions: 5,
		},
	}
	logger := zerolog.New(nil)

	riskManager := New("test_exchange", cfg, db, logger)
	return riskManager, db
}

func TestDefaultRiskConfig(t *testing.T) {
	config := defaultRiskConfig()

	if config == nil {
		t.Error("expected non-nil risk config")
	}

	// Test default values
	if config.MaxPositionSize != 0.1 {
		t.Errorf("expected max position size 0.1, got %f", config.MaxPositionSize)
	}
	if config.MaxDailyLoss != 1000.0 {
		t.Errorf("expected max daily loss 1000.0, got %f", config.MaxDailyLoss)
	}
	if config.MaxDailyVolume != 2.0 {
		t.Errorf("expected max daily volume 2.0, got %f", config.MaxDailyVolume)
	}
	if config.MaxDailyRisk != 0.2 {
		t.Errorf("expected max daily risk 0.2, got %f", config.MaxDailyRisk)
	}
	if config.MaxDrawdown != 0.15 {
		t.Errorf("expected max drawdown 0.15, got %f", config.MaxDrawdown)
	}
	if config.MaxOpenPositions != 5 {
		t.Errorf("expected max open positions 5, got %d", config.MaxOpenPositions)
	}
	if config.MaxPortfolioRisk != 0.25 {
		t.Errorf("expected max portfolio risk 0.25, got %f", config.MaxPortfolioRisk)
	}
	if config.MaxCorrelation != 0.7 {
		t.Errorf("expected max correlation 0.7, got %f", config.MaxCorrelation)
	}
	if config.MaxLeverage != 3.0 {
		t.Errorf("expected max leverage 3.0, got %f", config.MaxLeverage)
	}
	if config.MaxDailyTrades != 20 {
		t.Errorf("expected max daily trades 20, got %d", config.MaxDailyTrades)
	}
	if config.MaxHourlyTrades != 5 {
		t.Errorf("expected max hourly trades 5, got %d", config.MaxHourlyTrades)
	}
	if config.VaRLimit != 0.05 {
		t.Errorf("expected VaR limit 0.05, got %f", config.VaRLimit)
	}
	if config.MaxDrawdownLimit != 0.20 {
		t.Errorf("expected max drawdown limit 0.20, got %f", config.MaxDrawdownLimit)
	}
	if config.ConcentrationLimit != 0.30 {
		t.Errorf("expected concentration limit 0.30, got %f", config.ConcentrationLimit)
	}
}

func TestNew(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDatabase(t)
	defer db.Close()
	logger := zerolog.New(nil)

	riskManager := New("bybit", cfg, db, logger)

	if riskManager == nil {
		t.Error("expected non-nil risk manager")
	}
	if riskManager.exchangeName != "bybit" {
		t.Errorf("expected exchange name 'bybit', got '%s'", riskManager.exchangeName)
	}
	if riskManager.config != cfg {
		t.Error("expected config to be set")
	}
	if riskManager.db != db {
		t.Error("expected database to be set")
	}
	if riskManager.orderHistory == nil {
		t.Error("expected order history to be initialized")
	}
	if riskManager.dailyVolume == nil {
		t.Error("expected daily volume map to be initialized")
	}
	if riskManager.riskConfig == nil {
		t.Error("expected risk config to be initialized")
	}
}

func TestSetSettingsActor(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	// Mock PID (we can't create real PIDs easily in tests)
	// Just test that the function doesn't panic
	riskManager.SetSettingsActor(nil)
	
	if riskManager.settingsPID != nil {
		t.Error("expected settings PID to be nil when passing nil")
	}
}

func TestValidateOrderMsg(t *testing.T) {
	msg := ValidateOrderMsg{
		Exchange: "bybit",
		Symbol:   "BTCUSDT",
		Side:     "buy",
		Quantity: 1.0,
		Price:    50000.0,
	}

	if msg.Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", msg.Exchange)
	}
	if msg.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", msg.Symbol)
	}
	if msg.Side != "buy" {
		t.Errorf("expected side buy, got %s", msg.Side)
	}
	if msg.Quantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", msg.Quantity)
	}
	if msg.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", msg.Price)
	}
}

func TestOrderValidationResponse(t *testing.T) {
	response := OrderValidationResponse{
		Approved: true,
		Reason:   "Order approved",
		Warnings: []string{"High volatility", "Large position"},
	}

	if !response.Approved {
		t.Error("expected approved to be true")
	}
	if response.Reason != "Order approved" {
		t.Errorf("expected reason 'Order approved', got '%s'", response.Reason)
	}
	if len(response.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(response.Warnings))
	}
	if response.Warnings[0] != "High volatility" {
		t.Errorf("expected first warning 'High volatility', got '%s'", response.Warnings[0])
	}
	if response.Warnings[1] != "Large position" {
		t.Errorf("expected second warning 'Large position', got '%s'", response.Warnings[1])
	}
}

func TestOrderHistory(t *testing.T) {
	now := time.Now()
	history := OrderHistory{
		Timestamp: now,
		Exchange:  "bybit",
		Symbol:    "BTCUSDT",
		Side:      "buy",
		Quantity:  1.0,
		Price:     50000.0,
		Value:     50000.0,
	}

	if !history.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, history.Timestamp)
	}
	if history.Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", history.Exchange)
	}
	if history.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", history.Symbol)
	}
	if history.Side != "buy" {
		t.Errorf("expected side buy, got %s", history.Side)
	}
	if history.Quantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", history.Quantity)
	}
	if history.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", history.Price)
	}
	if history.Value != 50000.0 {
		t.Errorf("expected value 50000.0, got %f", history.Value)
	}
}

func TestPositionRisk(t *testing.T) {
	positionRisk := PositionRisk{
		Symbol:        "ETHUSDT",
		Value:         10000.0,
		Concentration: 0.2,
		VaR:           500.0,
	}

	if positionRisk.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", positionRisk.Symbol)
	}
	if positionRisk.Value != 10000.0 {
		t.Errorf("expected value 10000.0, got %f", positionRisk.Value)
	}
	if positionRisk.Concentration != 0.2 {
		t.Errorf("expected concentration 0.2, got %f", positionRisk.Concentration)
	}
	if positionRisk.VaR != 500.0 {
		t.Errorf("expected VaR 500.0, got %f", positionRisk.VaR)
	}
}

func TestValidateOrder(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	// Set portfolio value for testing
	riskManager.portfolioValue = 100000.0
	riskManager.cash = 50000.0
	riskManager.highWaterMark = 100000.0

	t.Run("approves valid order", func(t *testing.T) {
		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 0.1,
			Price:    50000.0,
		}

		response := riskManager.validateOrder(msg)

		if !response.Approved {
			t.Errorf("expected order to be approved, got reason: %s", response.Reason)
		}
	})

	t.Run("rejects order exceeding position size limit", func(t *testing.T) {
		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 1.0,
			Price:    50000.0, // Order value = 50000, max position = 10000 (10% of 100000)
		}

		response := riskManager.validateOrder(msg)

		if response.Approved {
			t.Error("expected order to be rejected for exceeding position size limit")
		}
		if response.Reason == "" {
			t.Error("expected rejection reason to be provided")
		}
	})

	t.Run("rejects buy order with insufficient cash", func(t *testing.T) {
		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 1.5,
			Price:    50000.0, // Order value = 75000, available cash = 50000
		}

		response := riskManager.validateOrder(msg)

		if response.Approved {
			t.Error("expected order to be rejected for insufficient cash")
		}
		if response.Reason == "" {
			t.Error("expected rejection reason to be provided")
		}
	})

	t.Run("rejects order exceeding daily risk limit", func(t *testing.T) {
		// Set daily risk used close to limit
		riskManager.dailyRiskUsed = 19000.0 // Close to 20000 (20% of 100000)

		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 0.1,
			Price:    20000.0, // Order value = 2000, would exceed daily risk limit
		}

		response := riskManager.validateOrder(msg)

		if response.Approved {
			t.Error("expected order to be rejected for exceeding daily risk limit")
		}
		if response.Reason == "" {
			t.Error("expected rejection reason to be provided")
		}

		// Reset for other tests
		riskManager.dailyRiskUsed = 0
	})

	t.Run("rejects order when drawdown exceeds limit", func(t *testing.T) {
		// Set portfolio value below drawdown limit
		riskManager.portfolioValue = 80000.0 // 20% drawdown from 100000
		riskManager.highWaterMark = 100000.0

		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 0.05,
			Price:    40000.0,
		}

		response := riskManager.validateOrder(msg)

		if response.Approved {
			t.Error("expected order to be rejected for excessive drawdown")
		}
		if response.Reason == "" {
			t.Error("expected rejection reason to be provided")
		}

		// Reset for other tests
		riskManager.portfolioValue = 100000.0
	})

	t.Run("provides warnings for orders close to limits", func(t *testing.T) {
		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 0.17,
			Price:    50000.0, // Order value = 8500, > 80% of position limit (8000)
		}

		response := riskManager.validateOrder(msg)

		if !response.Approved {
			t.Errorf("expected order to be approved, got reason: %s", response.Reason)
		}
		if len(response.Warnings) == 0 {
			t.Error("expected warnings for order close to limits")
		}
	})

	t.Run("handles daily volume limit", func(t *testing.T) {
		// Set daily volume close to limit
		today := time.Now().Format("2006-01-02")
		riskManager.dailyVolume[today] = 195000.0 // Close to 200000 (200% of 100000)

		msg := ValidateOrderMsg{
			Exchange: "test_exchange",
			Symbol:   "BTCUSDT",
			Side:     "buy",
			Quantity: 0.2,
			Price:    30000.0, // Order value = 6000, would exceed daily volume limit
		}

		response := riskManager.validateOrder(msg)

		if response.Approved {
			t.Error("expected order to be rejected for exceeding daily volume limit")
		}
		if response.Reason == "" {
			t.Error("expected rejection reason to be provided")
		}

		// Reset for other tests
		delete(riskManager.dailyVolume, today)
	})
}

func TestCalculatePositionConcentration(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	concentration := riskManager.calculatePositionConcentration()

	if concentration == nil {
		t.Error("expected non-nil concentration map")
	}

	// Test default values from the method
	if concentration["BTCUSDT"] != 0.25 {
		t.Errorf("expected BTCUSDT concentration 0.25, got %f", concentration["BTCUSDT"])
	}
	if concentration["ETHUSDT"] != 0.15 {
		t.Errorf("expected ETHUSDT concentration 0.15, got %f", concentration["ETHUSDT"])
	}
	if concentration["CASH"] != 0.60 {
		t.Errorf("expected CASH concentration 0.60, got %f", concentration["CASH"])
	}
}

func TestCalculateLeverageRatio(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	leverage := riskManager.calculateLeverageRatio()

	// Default implementation returns 1.0 (no leverage)
	if leverage != 1.0 {
		t.Errorf("expected leverage ratio 1.0, got %f", leverage)
	}
}

func TestCalculateVaR95(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	riskManager.portfolioValue = 100000.0

	t.Run("returns default VaR with insufficient history", func(t *testing.T) {
		var95 := riskManager.calculateVaR95()

		expected := 100000.0 * 0.05 // 5% of portfolio
		if var95 != expected {
			t.Errorf("expected VaR95 %f, got %f", expected, var95)
		}
	})

	t.Run("calculates VaR with sufficient history", func(t *testing.T) {
		// Add order history with varying values
		now := time.Now()
		for i := 0; i < 35; i++ {
			riskManager.orderHistory = append(riskManager.orderHistory, OrderHistory{
				Timestamp: now.Add(-time.Duration(i) * time.Hour),
				Value:     float64(1000 + i*100),
			})
		}

		var95 := riskManager.calculateVaR95()

		// Should calculate based on order history
		if var95 <= 0 {
			t.Error("expected positive VaR95 with order history")
		}
		if math.IsNaN(var95) || math.IsInf(var95, 0) {
			t.Error("expected finite VaR95 value")
		}
	})
}

func TestGetOrdersToday(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Add orders from today and yesterday
	riskManager.orderHistory = []OrderHistory{
		{Timestamp: now, Value: 1000.0},
		{Timestamp: now.Add(-time.Hour), Value: 2000.0},
		{Timestamp: yesterday, Value: 3000.0},
	}

	count := riskManager.getOrdersToday()

	if count != 2 {
		t.Errorf("expected 2 orders today, got %d", count)
	}
}

func TestResetDailyCounters(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	// Set some values
	riskManager.dailyRiskUsed = 5000.0
	
	// Add old daily volume data
	oldDate := time.Now().AddDate(0, 0, -40).Format("2006-01-02")
	recentDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	riskManager.dailyVolume[oldDate] = 10000.0
	riskManager.dailyVolume[recentDate] = 20000.0

	riskManager.resetDailyCounters()

	if riskManager.dailyRiskUsed != 0 {
		t.Errorf("expected daily risk used to be reset to 0, got %f", riskManager.dailyRiskUsed)
	}

	// Old data should be cleaned up
	if _, exists := riskManager.dailyVolume[oldDate]; exists {
		t.Error("expected old daily volume data to be cleaned up")
	}

	// Recent data should remain
	if _, exists := riskManager.dailyVolume[recentDate]; !exists {
		t.Error("expected recent daily volume data to remain")
	}
}

func TestUpdateRiskMetrics(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	// Add many orders to test cleanup
	for i := 0; i < 1200; i++ {
		riskManager.orderHistory = append(riskManager.orderHistory, OrderHistory{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Value:     float64(i * 100),
		})
	}

	originalLength := len(riskManager.orderHistory)
	riskManager.updateRiskMetrics()

	// Should keep only last 1000 orders
	if len(riskManager.orderHistory) != 1000 {
		t.Errorf("expected order history to be trimmed to 1000, got %d", len(riskManager.orderHistory))
	}

	if len(riskManager.orderHistory) >= originalLength {
		t.Error("expected order history to be reduced")
	}
}

func TestRiskMetricsResponse(t *testing.T) {
	response := RiskMetricsResponse{
		MaxDrawdown:           0.15,
		VaR95:                 5000.0,
		PositionConcentration: map[string]float64{"BTC": 0.5},
		LeverageRatio:         1.2,
		DailyRiskLimit:        20000.0,
		DailyRiskUsed:         5000.0,
	}

	if response.MaxDrawdown != 0.15 {
		t.Errorf("expected max drawdown 0.15, got %f", response.MaxDrawdown)
	}
	if response.VaR95 != 5000.0 {
		t.Errorf("expected VaR95 5000.0, got %f", response.VaR95)
	}
	if response.PositionConcentration["BTC"] != 0.5 {
		t.Errorf("expected BTC concentration 0.5, got %f", response.PositionConcentration["BTC"])
	}
	if response.LeverageRatio != 1.2 {
		t.Errorf("expected leverage ratio 1.2, got %f", response.LeverageRatio)
	}
	if response.DailyRiskLimit != 20000.0 {
		t.Errorf("expected daily risk limit 20000.0, got %f", response.DailyRiskLimit)
	}
	if response.DailyRiskUsed != 5000.0 {
		t.Errorf("expected daily risk used 5000.0, got %f", response.DailyRiskUsed)
	}
}

func TestUpdateLocalRiskConfig(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	tests := []struct {
		name      string
		parameter string
		value     interface{}
		check     func() bool
	}{
		{
			name:      "updates max position size with float64",
			parameter: "max_position_size",
			value:     0.2,
			check:     func() bool { return riskManager.riskConfig.MaxPositionSize == 0.2 },
		},
		{
			name:      "updates max position size with string",
			parameter: "max_position_size",
			value:     "0.25",
			check:     func() bool { return riskManager.riskConfig.MaxPositionSize == 0.25 },
		},
		{
			name:      "updates max daily loss",
			parameter: "max_daily_loss",
			value:     2000.0,
			check:     func() bool { return riskManager.riskConfig.MaxDailyLoss == 2000.0 },
		},
		{
			name:      "updates max daily trades with int",
			parameter: "max_daily_trades",
			value:     30,
			check:     func() bool { return riskManager.riskConfig.MaxDailyTrades == 30 },
		},
		{
			name:      "updates max daily trades with string",
			parameter: "max_daily_trades",
			value:     "25",
			check:     func() bool { return riskManager.riskConfig.MaxDailyTrades == 25 },
		},
		{
			name:      "ignores invalid parameter",
			parameter: "invalid_parameter",
			value:     123,
			check:     func() bool { return true }, // Should not change anything
		},
		{
			name:      "ignores invalid value type",
			parameter: "max_position_size",
			value:     []int{1, 2, 3}, // Invalid type
			check:     func() bool { return riskManager.riskConfig.MaxPositionSize == 0.25 }, // Should remain unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskManager.updateLocalRiskConfig(tt.parameter, tt.value)
			if !tt.check() {
				t.Errorf("update failed for parameter %s with value %v", tt.parameter, tt.value)
			}
		})
	}
}

func TestGetLocalRiskParameter(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	// Set a known value
	riskManager.riskConfig.MaxPositionSize = 0.12

	tests := []struct {
		name      string
		parameter string
		expected  interface{}
	}{
		{
			name:      "gets max position size",
			parameter: "max_position_size",
			expected:  0.12,
		},
		{
			name:      "gets max daily loss",
			parameter: "max_daily_loss",
			expected:  riskManager.riskConfig.MaxDailyLoss,
		},
		{
			name:      "gets max daily trades",
			parameter: "max_daily_trades",
			expected:  riskManager.riskConfig.MaxDailyTrades,
		},
		{
			name:      "returns nil for unknown parameter",
			parameter: "unknown_parameter",
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := riskManager.getLocalRiskParameter(tt.parameter)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestUpdateLocalRiskConfigFromString(t *testing.T) {
	riskManager, db := setupTestRiskManager(t)
	defer db.Close()

	tests := []struct {
		name      string
		parameter string
		value     string
		check     func() bool
	}{
		{
			name:      "updates float parameter from string",
			parameter: "max_position_size",
			value:     "0.3",
			check:     func() bool { return riskManager.riskConfig.MaxPositionSize == 0.3 },
		},
		{
			name:      "updates int parameter from string",
			parameter: "max_daily_trades",
			value:     "35",
			check:     func() bool { return riskManager.riskConfig.MaxDailyTrades == 35 },
		},
		{
			name:      "ignores invalid float string",
			parameter: "max_leverage",
			value:     "invalid_float",
			check:     func() bool { return riskManager.riskConfig.MaxLeverage == 3.0 }, // Should remain default
		},
		{
			name:      "ignores invalid int string",
			parameter: "max_hourly_trades",
			value:     "invalid_int",
			check:     func() bool { return riskManager.riskConfig.MaxHourlyTrades == 5 }, // Should remain default
		},
		{
			name:      "ignores unknown parameter",
			parameter: "unknown_param",
			value:     "123",
			check:     func() bool { return true }, // Should not panic or change anything
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskManager.updateLocalRiskConfigFromString(tt.parameter, tt.value)
			if !tt.check() {
				t.Errorf("update failed for parameter %s with value %s", tt.parameter, tt.value)
			}
		})
	}
}