package strategy

import (
	"math"
	"testing"
)

func TestNewIndicators(t *testing.T) {
	ti := &TechnicalIndicators{}

	// Test data
	highs := []float64{105, 106, 107, 108, 109, 108, 107, 106, 105, 104, 103, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	lows := []float64{95, 96, 97, 98, 99, 98, 97, 96, 95, 94, 93, 92, 93, 94, 95, 96, 97, 98, 99, 100}
	closes := []float64{100, 101, 102, 103, 104, 103, 102, 101, 100, 99, 98, 97, 98, 99, 100, 101, 102, 103, 104, 105}
	opens := []float64{99, 100, 101, 102, 103, 102, 101, 100, 99, 98, 97, 96, 97, 98, 99, 100, 101, 102, 103, 104}
	volumes := []float64{1000, 1100, 1200, 1300, 1400, 1300, 1200, 1100, 1000, 900, 800, 700, 800, 900, 1000, 1100, 1200, 1300, 1400, 1500}

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			"Hull Moving Average",
			func(t *testing.T) {
				result := ti.calculateHullMA(createStarlarkList(closes), 10)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
				
				// Check that we get valid values after warm-up period
				validCount := 0
				for _, val := range result {
					if !math.IsNaN(val) {
						validCount++
					}
				}
				if validCount == 0 {
					t.Error("Expected some valid HMA values")
				}
			},
		},
		{
			"Weighted Moving Average",
			func(t *testing.T) {
				result := ti.calculateWMA(createStarlarkList(closes), 5)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
				
				// First 4 values should be NaN
				for i := 0; i < 4; i++ {
					if !math.IsNaN(result[i]) {
						t.Errorf("Expected NaN at index %d, got %f", i, result[i])
					}
				}
			},
		},
		{
			"Chandelier Exit",
			func(t *testing.T) {
				longExit, shortExit := ti.calculateChandelierExit(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), 10, 3.0)
				if len(longExit) != len(closes) || len(shortExit) != len(closes) {
					t.Errorf("Expected %d results for both exits, got %d and %d", len(closes), len(longExit), len(shortExit))
				}
			},
		},
		{
			"ALMA",
			func(t *testing.T) {
				result := ti.calculateALMA(createStarlarkList(closes), 10, 0.85, 6.0)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"CMO",
			func(t *testing.T) {
				result := ti.calculateCMO(createStarlarkList(closes), 14)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
				
				// Check that values are within expected range
				for i, val := range result {
					if !math.IsNaN(val) && (val < -100 || val > 100) {
						t.Errorf("CMO value at index %d out of range [-100, 100]: %f", i, val)
					}
				}
			},
		},
		{
			"TEMA",
			func(t *testing.T) {
				result := ti.calculateTEMA(createStarlarkList(closes), 5)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"EMV",
			func(t *testing.T) {
				result := ti.calculateEMV(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), createStarlarkList(volumes), 14)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"Force Index",
			func(t *testing.T) {
				result := ti.calculateForceIndex(createStarlarkList(closes), createStarlarkList(volumes), 13)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"Balance of Power",
			func(t *testing.T) {
				result := ti.calculateBOP(createStarlarkList(opens), createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes))
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
				
				// Check that values are within expected range
				for i, val := range result {
					if val < -1 || val > 1 {
						t.Errorf("BOP value at index %d out of range [-1, 1]: %f", i, val)
					}
				}
			},
		},
		{
			"Price Channel",
			func(t *testing.T) {
				upper, middle, lower := ti.calculatePriceChannel(createStarlarkList(highs), createStarlarkList(lows), 10)
				if len(upper) != len(closes) || len(middle) != len(closes) || len(lower) != len(closes) {
					t.Errorf("Expected %d results for all channels", len(closes))
				}
				
				// Check that upper >= middle >= lower when valid
				for i := range upper {
					if !math.IsNaN(upper[i]) && !math.IsNaN(middle[i]) && !math.IsNaN(lower[i]) {
						if upper[i] < middle[i] || middle[i] < lower[i] {
							t.Errorf("Price channel ordering violation at index %d: upper=%f, middle=%f, lower=%f", i, upper[i], middle[i], lower[i])
						}
					}
				}
			},
		},
		{
			"Mass Index",
			func(t *testing.T) {
				result := ti.calculateMassIndex(createStarlarkList(highs), createStarlarkList(lows), 9, 25)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"Volume Oscillator",
			func(t *testing.T) {
				result := ti.calculateVolumeOscillator(createStarlarkList(volumes), 5, 10)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"KST",
			func(t *testing.T) {
				kst, signal := ti.calculateKST(createStarlarkList(closes), 10, 15, 20, 30, 10, 10, 10, 15)
				if len(kst) != len(closes) || len(signal) != len(closes) {
					t.Errorf("Expected %d results for both KST and signal", len(closes))
				}
			},
		},
		{
			"STC",
			func(t *testing.T) {
				result := ti.calculateSTC(createStarlarkList(closes), 5, 10, 5, 0.5)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
				
				// Check that values are within expected range (0-100)
				for i, val := range result {
					if !math.IsNaN(val) && (val < 0 || val > 100) {
						t.Errorf("STC value at index %d out of range [0, 100]: %f", i, val)
					}
				}
			},
		},
		{
			"Coppock Curve",
			func(t *testing.T) {
				result := ti.calculateCoppockCurve(createStarlarkList(closes), 14, 11, 10)
				if len(result) != len(closes) {
					t.Errorf("Expected %d results, got %d", len(closes), len(result))
				}
			},
		},
		{
			"Chande Kroll Stop",
			func(t *testing.T) {
				longStop, shortStop := ti.calculateChandeKrollStop(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), 10, 3.0)
				if len(longStop) != len(closes) || len(shortStop) != len(closes) {
					t.Errorf("Expected %d results for both stops", len(closes))
				}
			},
		},
		{
			"Elder Force Index",
			func(t *testing.T) {
				shortFI, longFI := ti.calculateElderForceIndex(createStarlarkList(closes), createStarlarkList(volumes), 2, 13)
				if len(shortFI) != len(closes) || len(longFI) != len(closes) {
					t.Errorf("Expected %d results for both force indices", len(closes))
				}
			},
		},
		{
			"Klinger Oscillator",
			func(t *testing.T) {
				ko, signal := ti.calculateKlingerOscillator(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), createStarlarkList(volumes), 34, 55, 13)
				if len(ko) != len(closes) || len(signal) != len(closes) {
					t.Errorf("Expected %d results for both oscillator and signal", len(closes))
				}
			},
		},
		{
			"Volume Profile",
			func(t *testing.T) {
				profile := ti.calculateVolumeProfile(createStarlarkList(highs), createStarlarkList(lows), createStarlarkList(closes), createStarlarkList(volumes), 20, 10)
				if profile == nil {
					t.Error("Expected volume profile map, got nil")
				}
				
				if len(profile) == 0 {
					t.Error("Expected volume profile entries, got empty map")
				}
				
				// Check that all volume values are non-negative
				for price, vol := range profile {
					if vol < 0 {
						t.Errorf("Negative volume at price %f: %f", price, vol)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestNewIndicatorValues(t *testing.T) {
	ti := &TechnicalIndicators{}

	// Test with simple trending data
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	priceList := createStarlarkList(prices)

	t.Run("Hull MA structure validation", func(t *testing.T) {
		result := ti.calculateHullMA(priceList, 5)
		if result == nil {
			t.Fatal("Hull MA calculation returned nil")
		}

		if len(result) != len(prices) {
			t.Errorf("Expected %d values, got %d", len(prices), len(result))
		}

		// Should have some valid values
		validCount := 0
		for _, val := range result {
			if !math.IsNaN(val) {
				validCount++
			}
		}

		if validCount == 0 {
			t.Error("Expected some valid Hull MA values")
		}

		t.Logf("Hull MA structure validation passed")
	})

	t.Run("CMO range validation", func(t *testing.T) {
		result := ti.calculateCMO(priceList, 5)
		if result == nil {
			t.Fatal("CMO calculation returned nil")
		}

		for i, val := range result {
			if !math.IsNaN(val) {
				if val < -100 || val > 100 {
					t.Errorf("CMO value at index %d out of expected range [-100, 100]: %f", i, val)
				}
			}
		}

		t.Logf("CMO range validation passed")
	})
}