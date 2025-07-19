package strategy

import (
	"math"
	"testing"

	"go.starlark.net/starlark"
)

// Helper function to create a starlark list from float slice
func createStarlarkList(values []float64) *starlark.List {
	list := starlark.NewList(nil)
	for _, v := range values {
		list.Append(starlark.Float(v))
	}
	return list
}

func TestWilliamsR(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data
	highs := []float64{105, 106, 107, 108, 109, 108, 107, 106, 105, 104, 103, 102, 103, 104, 105}
	lows := []float64{95, 96, 97, 98, 99, 98, 97, 96, 95, 94, 93, 92, 93, 94, 95}
	closes := []float64{100, 101, 102, 103, 104, 103, 102, 101, 100, 99, 98, 97, 98, 99, 100}
	
	result := ti.calculateWilliamsR(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), 14)
	
	if len(result) != len(closes) {
		t.Errorf("Expected %d results, got %d", len(closes), len(result))
	}
	
	// First 13 values should be NaN
	for i := 0; i < 13; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, result[i])
		}
	}
	
	// Last value should be a valid Williams %R value (between -100 and 0)
	lastValue := result[len(result)-1]
	if math.IsNaN(lastValue) || lastValue > 0 || lastValue < -100 {
		t.Errorf("Expected Williams %%R between -100 and 0, got %f", lastValue)
	}
}

func TestATR(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data with some volatility
	highs := []float64{105, 110, 115, 108, 112, 109, 107, 111, 105, 103, 108, 102, 106, 104, 109}
	lows := []float64{95, 90, 85, 92, 88, 91, 93, 89, 95, 97, 92, 98, 94, 96, 91}
	closes := []float64{100, 95, 90, 95, 90, 95, 100, 95, 100, 99, 95, 100, 99, 100, 95}
	
	result := ti.calculateATR(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), 14)
	
	if len(result) != len(closes) {
		t.Errorf("Expected %d results, got %d", len(closes), len(result))
	}
	
	// First 14 values should be NaN (including index 0)
	for i := 0; i < 14; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, result[i])
		}
	}
	
	// Last value should be a positive ATR value
	lastValue := result[len(result)-1]
	if math.IsNaN(lastValue) || lastValue <= 0 {
		t.Errorf("Expected positive ATR value, got %f", lastValue)
	}
}

func TestCCI(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data
	highs := []float64{105, 106, 107, 108, 109, 108, 107, 106, 105, 104, 103, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	lows := []float64{95, 96, 97, 98, 99, 98, 97, 96, 95, 94, 93, 92, 93, 94, 95, 96, 97, 98, 99, 100}
	closes := []float64{100, 101, 102, 103, 104, 103, 102, 101, 100, 99, 98, 97, 98, 99, 100, 101, 102, 103, 104, 105}
	
	result := ti.calculateCCI(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), 20)
	
	if len(result) != len(closes) {
		t.Errorf("Expected %d results, got %d", len(closes), len(result))
	}
	
	// First 19 values should be NaN
	for i := 0; i < 19; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, result[i])
		}
	}
	
	// Last value should be a valid CCI value
	lastValue := result[len(result)-1]
	if math.IsNaN(lastValue) {
		t.Errorf("Expected valid CCI value, got NaN")
	}
}

func TestVWAP(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data
	highs := []float64{105, 106, 107, 108, 109}
	lows := []float64{95, 96, 97, 98, 99}
	closes := []float64{100, 101, 102, 103, 104}
	volumes := []float64{1000, 1100, 1200, 1300, 1400}
	
	result := ti.calculateVWAP(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), createStarlarkList(volumes))
	
	if len(result) != len(closes) {
		t.Errorf("Expected %d results, got %d", len(closes), len(result))
	}
	
	// All values should be valid (VWAP is cumulative)
	for i, val := range result {
		if math.IsNaN(val) || val <= 0 {
			t.Errorf("Expected positive VWAP value at index %d, got %f", i, val)
		}
	}
	
	// VWAP should be around the typical price range
	lastValue := result[len(result)-1]
	if lastValue < 95 || lastValue > 110 {
		t.Errorf("VWAP value seems out of range: %f", lastValue)
	}
}

func TestMFI(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data with varying volumes
	highs := []float64{105, 106, 107, 108, 109, 108, 107, 106, 105, 104, 103, 102, 103, 104, 105, 106}
	lows := []float64{95, 96, 97, 98, 99, 98, 97, 96, 95, 94, 93, 92, 93, 94, 95, 96}
	closes := []float64{100, 101, 102, 103, 104, 103, 102, 101, 100, 99, 98, 97, 98, 99, 100, 101}
	volumes := []float64{1000, 1100, 1200, 1300, 1400, 1350, 1250, 1150, 1050, 950, 850, 750, 850, 950, 1050, 1150}
	
	result := ti.calculateMFI(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), createStarlarkList(volumes), 14)
	
	if len(result) != len(closes) {
		t.Errorf("Expected %d results, got %d", len(closes), len(result))
	}
	
	// First 14 values should be NaN
	for i := 0; i < 14; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, result[i])
		}
	}
	
	// Valid MFI values should be between 0 and 100
	for i := 14; i < len(result); i++ {
		val := result[i]
		if math.IsNaN(val) || val < 0 || val > 100 {
			t.Errorf("Expected MFI between 0-100 at index %d, got %f", i, val)
		}
	}
}

func TestStdDev(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data with known standard deviation
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	
	result := ti.calculateStdDev(createStarlarkList(prices), 10)
	
	if len(result) != len(prices) {
		t.Errorf("Expected %d results, got %d", len(prices), len(result))
	}
	
	// First 9 values should be NaN
	for i := 0; i < 9; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, result[i])
		}
	}
	
	// Last values should be positive
	for i := 9; i < len(result); i++ {
		val := result[i]
		if math.IsNaN(val) || val < 0 {
			t.Errorf("Expected positive standard deviation at index %d, got %f", i, val)
		}
	}
}

func TestROC(t *testing.T) {
	ti := &TechnicalIndicators{}
	
	// Test data with known rate of change
	prices := []float64{100, 102, 104, 106, 108, 110, 112, 114, 116, 118, 120}
	
	result := ti.calculateROC(createStarlarkList(prices), 5)
	
	if len(result) != len(prices) {
		t.Errorf("Expected %d results, got %d", len(prices), len(result))
	}
	
	// First 5 values should be NaN
	for i := 0; i < 5; i++ {
		if !math.IsNaN(result[i]) {
			t.Errorf("Expected NaN at index %d, got %f", i, result[i])
		}
	}
	
	// ROC values should be reasonable percentages
	for i := 5; i < len(result); i++ {
		val := result[i]
		if math.IsNaN(val) {
			t.Errorf("Expected valid ROC value at index %d, got NaN", i)
		}
		// For our test data, ROC should be positive (prices are increasing)
		if val <= 0 {
			t.Errorf("Expected positive ROC for increasing prices at index %d, got %f", i, val)
		}
	}
}