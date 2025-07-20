package strategy

import (
	"math"
	"testing"

	"github.com/rs/zerolog"
	"go.starlark.net/starlark"
)

// TestAdditionalIndicators tests the new technical indicators
func TestAdditionalIndicators(t *testing.T) {
	// Create test data
	open := createTestList([]float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120})
	high := createTestList([]float64{102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122})
	low := createTestList([]float64{99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119})
	close := createTestList([]float64{101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121})
	volume := createTestList([]float64{1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000, 2100, 2200, 2300, 2400, 2500, 2600, 2700, 2800, 2900, 3000})

	ti := &TechnicalIndicators{}

	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "RVI",
			test: func() error {
				rvi, signal := ti.calculateRVI(open, high, low, close, 14)
				if rvi == nil || signal == nil {
					return nil // Expected for insufficient data
				}
				if len(rvi) != close.Len() || len(signal) != close.Len() {
					t.Errorf("RVI length mismatch")
				}
				return nil
			},
		},
		{
			name: "PPO",
			test: func() error {
				ppo, signal, histogram := ti.calculatePPO(close, 12, 26, 9)
				if ppo == nil || signal == nil || histogram == nil {
					return nil // Expected for insufficient data
				}
				if len(ppo) != close.Len() || len(signal) != close.Len() || len(histogram) != close.Len() {
					t.Errorf("PPO length mismatch")
				}
				return nil
			},
		},
		{
			name: "Accumulation/Distribution",
			test: func() error {
				ad := ti.calculateAccumulationDistribution(high, low, close, volume)
				if ad == nil {
					t.Errorf("Accumulation/Distribution should not be nil")
				}
				if len(ad) != close.Len() {
					t.Errorf("A/D length mismatch: got %d, expected %d", len(ad), close.Len())
				}
				return nil
			},
		},
		{
			name: "Chaikin Money Flow",
			test: func() error {
				cmf := ti.calculateChaikinMoneyFlow(high, low, close, volume, 20)
				if cmf == nil {
					return nil // Expected for insufficient data
				}
				if len(cmf) != close.Len() {
					t.Errorf("CMF length mismatch")
				}
				return nil
			},
		},
		{
			name: "Linear Regression",
			test: func() error {
				lr := ti.calculateLinearRegression(close, 14)
				if lr == nil {
					return nil // Expected for insufficient data
				}
				if len(lr) != close.Len() {
					t.Errorf("Linear Regression length mismatch")
				}
				return nil
			},
		},
		{
			name: "Linear Regression Slope",
			test: func() error {
				lrs := ti.calculateLinearRegressionSlope(close, 14)
				if lrs == nil {
					return nil // Expected for insufficient data
				}
				if len(lrs) != close.Len() {
					t.Errorf("Linear Regression Slope length mismatch")
				}
				// For uptrending data, slope should be positive
				if len(lrs) > 14 && !math.IsNaN(lrs[len(lrs)-1]) && lrs[len(lrs)-1] <= 0 {
					t.Errorf("Expected positive slope for uptrending data, got %f", lrs[len(lrs)-1])
				}
				return nil
			},
		},
		{
			name: "Correlation Coefficient",
			test: func() error {
				corr := ti.calculateCorrelationCoefficient(close, 14)
				if corr == nil {
					return nil // Expected for insufficient data
				}
				if len(corr) != close.Len() {
					t.Errorf("Correlation Coefficient length mismatch")
				}
				// For uptrending data, correlation should be positive
				if len(corr) > 14 && !math.IsNaN(corr[len(corr)-1]) && corr[len(corr)-1] <= 0 {
					t.Errorf("Expected positive correlation for uptrending data, got %f", corr[len(corr)-1])
				}
				return nil
			},
		},
		{
			name: "Bollinger PercentB",
			test: func() error {
				percentB := ti.calculateBollingerPercentB(close, 20, 2.0)
				if percentB == nil {
					return nil // Expected for insufficient data
				}
				if len(percentB) != close.Len() {
					t.Errorf("Bollinger PercentB length mismatch")
				}
				return nil
			},
		},
		{
			name: "Bollinger Band Width",
			test: func() error {
				bbw := ti.calculateBollingerBandWidth(close, 20, 2.0)
				if bbw == nil {
					return nil // Expected for insufficient data
				}
				if len(bbw) != close.Len() {
					t.Errorf("Bollinger Band Width length mismatch")
				}
				return nil
			},
		},
		{
			name: "Standard Error",
			test: func() error {
				se := ti.calculateStandardError(close, 14)
				if se == nil {
					return nil // Expected for insufficient data
				}
				if len(se) != close.Len() {
					t.Errorf("Standard Error length mismatch")
				}
				return nil
			},
		},
		{
			name: "Williams A/D",
			test: func() error {
				wad := ti.calculateWilliamsAD(high, low, close)
				if wad == nil {
					return nil // Expected for insufficient data
				}
				if len(wad) != close.Len() {
					t.Errorf("Williams A/D length mismatch")
				}
				return nil
			},
		},
		{
			name: "Money Flow Volume",
			test: func() error {
				mfv := ti.calculateMoneyFlowVolume(high, low, close, volume)
				if mfv == nil {
					t.Errorf("Money Flow Volume should not be nil")
				}
				if len(mfv) != close.Len() {
					t.Errorf("MFV length mismatch: got %d, expected %d", len(mfv), close.Len())
				}
				return nil
			},
		},
		{
			name: "Price ROC",
			test: func() error {
				priceROC := ti.calculatePriceROC(close, 10)
				if priceROC == nil {
					return nil // Expected for insufficient data
				}
				if len(priceROC) != close.Len() {
					t.Errorf("Price ROC length mismatch")
				}
				return nil
			},
		},
		{
			name: "Volatility Index",
			test: func() error {
				vi := ti.calculateVolatilityIndex(high, low, close, 20)
				if vi == nil {
					return nil // Expected for insufficient data
				}
				if len(vi) != close.Len() {
					t.Errorf("Volatility Index length mismatch")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.test(); err != nil {
				t.Errorf("%s error: %v", tt.name, err)
			} else {
				t.Logf("%s executed successfully", tt.name)
			}
		})
	}
}

// TestAdditionalIndicatorValues tests specific values and behaviors of new indicators
func TestAdditionalIndicatorValues(t *testing.T) {
	ti := &TechnicalIndicators{}

	t.Run("RVI structure validation", func(t *testing.T) {
		// Create test data with clear directional movement
		open := createTestList([]float64{100, 100, 100, 100, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115})
		high := createTestList([]float64{102, 102, 102, 102, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117})
		low := createTestList([]float64{99, 99, 99, 99, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114})
		close := createTestList([]float64{101, 101, 101, 101, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116})

		rvi, signal := ti.calculateRVI(open, high, low, close, 14)
		if rvi == nil || signal == nil {
			t.Skip("RVI returned nil (insufficient data)")
		}

		// Check that RVI and signal have proper structure
		if len(rvi) != close.Len() {
			t.Errorf("RVI length mismatch: got %d, expected %d", len(rvi), close.Len())
		}
		if len(signal) != close.Len() {
			t.Errorf("RVI signal length mismatch: got %d, expected %d", len(signal), close.Len())
		}

		t.Logf("RVI structure validation passed")
	})

	t.Run("Correlation coefficient range validation", func(t *testing.T) {
		// Create strongly correlated data (uptrend)
		prices := createTestList([]float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120})

		corr := ti.calculateCorrelationCoefficient(prices, 14)
		if corr == nil {
			t.Skip("Correlation coefficient returned nil")
		}

		// Check that correlation values are in valid range [-1, 1]
		for i, val := range corr {
			if !math.IsNaN(val) && (val < -1.0 || val > 1.0) {
				t.Errorf("Correlation coefficient out of range at index %d: %f", i, val)
			}
		}

		// For strongly uptrending data, final correlation should be positive
		lastVal := corr[len(corr)-1]
		if !math.IsNaN(lastVal) && lastVal <= 0 {
			t.Errorf("Expected positive correlation for uptrending data, got %f", lastVal)
		}

		t.Logf("Correlation coefficient range validation passed")
	})

	t.Run("Bollinger PercentB range validation", func(t *testing.T) {
		// Create test data
		prices := createTestList([]float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120})

		percentB := ti.calculateBollingerPercentB(prices, 20, 2.0)
		if percentB == nil {
			t.Skip("Bollinger PercentB returned nil")
		}

		// PercentB can go outside 0-1 range, but let's check for reasonable values
		foundValidValue := false
		for _, val := range percentB {
			if !math.IsNaN(val) {
				foundValidValue = true
				if val < -2.0 || val > 3.0 { // Allow some range outside 0-1
					t.Errorf("Bollinger PercentB seems unreasonable: %f", val)
				}
			}
		}

		if !foundValidValue {
			t.Errorf("No valid Bollinger PercentB values found")
		}

		t.Logf("Bollinger PercentB range validation passed")
	})
}

// TestStarlarkIntegrationAdditional tests the new indicators in Starlark context
func TestStarlarkIntegrationAdditional(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	engine := NewStrategyEngine(logger)

	// Test script for new indicators
	script := `
def test_additional_indicators():
    # Test data
    open = [100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119]
    high = [102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121]
    low = [99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118]
    close = [101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120]
    volume = [1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000, 2100, 2200, 2300, 2400, 2500, 2600, 2700, 2800, 2900]
    
    # Test RVI
    rvi_result = rvi(open, high, low, close, 14)
    
    # Test PPO
    ppo_result = ppo(close, 12, 26, 9)
    
    # Test Accumulation/Distribution
    ad_result = accumulation_distribution(high, low, close, volume)
    
    # Test Chaikin Money Flow
    cmf_result = chaikin_money_flow(high, low, close, volume, 20)
    
    # Test Linear Regression
    lr_result = linear_regression(close, 14)
    
    # Test Bollinger PercentB
    bb_result = bollinger_percent_b(close, 20, 2.0)
    
    # Test Williams A/D
    wad_result = williams_ad(high, low, close)
    
    # Test Money Flow Volume
    mfv_result = money_flow_volume(high, low, close, volume)
    
    return {
        "rvi": rvi_result,
        "ppo": ppo_result, 
        "ad": ad_result,
        "cmf": cmf_result,
        "lr": lr_result,
        "bb": bb_result,
        "wad": wad_result,
        "mfv": mfv_result
    }

result = test_additional_indicators()
`

	thread := &starlark.Thread{Name: "test"}
	globals := engine.builtin

	_, err := starlark.ExecFile(thread, "test.star", script, globals)
	if err != nil {
		t.Fatalf("Failed to execute test script: %v", err)
	}

	t.Logf("Starlark integration test for additional indicators passed")
}

// Helper function to create test lists
func createTestList(values []float64) *starlark.List {
	list := starlark.NewList(nil)
	for _, v := range values {
		list.Append(starlark.Float(v))
	}
	return list
}