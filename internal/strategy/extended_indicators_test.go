package strategy

import (
	"testing"

	"github.com/rs/zerolog"
	"go.starlark.net/starlark"
)

// TestExtendedIndicators tests the newly added extended technical indicators
func TestExtendedIndicators(t *testing.T) {
	logger := zerolog.Nop()
	engine := NewStrategyEngine(logger)

	// Create test data
	prices := []float64{100, 102, 101, 103, 105, 104, 106, 108, 107, 109, 111, 110, 112, 114, 113, 115, 117, 116, 118, 120, 122, 121, 123, 125, 124, 126, 128, 127, 129, 131, 130, 132, 134, 133, 135}
	high := []float64{101, 103, 102, 104, 106, 105, 107, 109, 108, 110, 112, 111, 113, 115, 114, 116, 118, 117, 119, 121, 123, 122, 124, 126, 125, 127, 129, 128, 130, 132, 131, 133, 135, 134, 136}
	low := []float64{99, 101, 100, 102, 104, 103, 105, 107, 106, 108, 110, 109, 111, 113, 112, 114, 116, 115, 117, 119, 121, 120, 122, 124, 123, 125, 127, 126, 128, 130, 129, 131, 133, 132, 134}
	open := []float64{99.5, 101.5, 100.5, 102.5, 104.5, 103.5, 105.5, 107.5, 106.5, 108.5, 110.5, 109.5, 111.5, 113.5, 112.5, 114.5, 116.5, 115.5, 117.5, 119.5, 121.5, 120.5, 122.5, 124.5, 123.5, 125.5, 127.5, 126.5, 128.5, 130.5, 129.5, 131.5, 133.5, 132.5, 134.5}

	// Convert to Starlark lists
	pricesList := engine.floatListToStarlark(prices)
	highList := engine.floatListToStarlark(high)
	lowList := engine.floatListToStarlark(low)
	openList := engine.floatListToStarlark(open)

	tests := []struct {
		name        string
		function    func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)
		args        starlark.Tuple
		expectDict  bool
		expectedLen int
	}{
		{
			name:        "Heikin Ashi",
			function:    engine.heikinAshi,
			args:        starlark.Tuple{openList, highList, lowList, pricesList},
			expectDict:  true,
			expectedLen: 4, // open, high, low, close
		},
		{
			name:        "Vortex Indicator",
			function:    engine.vortex,
			args:        starlark.Tuple{highList, lowList, pricesList, starlark.MakeInt(14)},
			expectDict:  true,
			expectedLen: 2, // vi_plus, vi_minus
		},
		{
			name:        "Williams Alligator",
			function:    engine.williamsAlligator,
			args:        starlark.Tuple{pricesList},
			expectDict:  true,
			expectedLen: 3, // jaw, teeth, lips
		},
		{
			name:        "Supertrend",
			function:    engine.supertrend,
			args:        starlark.Tuple{highList, lowList, pricesList, starlark.MakeInt(10), starlark.Float(3.0)},
			expectDict:  true,
			expectedLen: 2, // supertrend, trend
		},
		{
			name:        "Stochastic RSI",
			function:    engine.stochasticRSI,
			args:        starlark.Tuple{pricesList, starlark.MakeInt(14), starlark.MakeInt(14), starlark.MakeInt(3), starlark.MakeInt(3)},
			expectDict:  true,
			expectedLen: 2, // k, d
		},
		{
			name:        "Awesome Oscillator",
			function:    engine.awesomeOscillator,
			args:        starlark.Tuple{highList, lowList},
			expectDict:  false,
			expectedLen: 35,
		},
		{
			name:        "Accelerator Oscillator",
			function:    engine.acceleratorOscillator,
			args:        starlark.Tuple{highList, lowList, pricesList},
			expectDict:  false,
			expectedLen: 35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			result, err := tt.function(nil, nil, tt.args, nil)
			if err != nil {
				t.Fatalf("Error calling %s: %v", tt.name, err)
			}

			// Check result type and length
			if tt.expectDict {
				dict, ok := result.(*starlark.Dict)
				if !ok {
					t.Fatalf("%s should return a dict, got %T", tt.name, result)
				}
				if dict.Len() != tt.expectedLen {
					t.Errorf("%s dict length = %d, want %d", tt.name, dict.Len(), tt.expectedLen)
				}
			} else {
				list, ok := result.(*starlark.List)
				if !ok {
					t.Fatalf("%s should return a list, got %T", tt.name, result)
				}
				if list.Len() != tt.expectedLen {
					t.Errorf("%s list length = %d, want %d", tt.name, list.Len(), tt.expectedLen)
				}
			}

			t.Logf("%s executed successfully", tt.name)
		})
	}
}

// TestIndicatorValues tests specific functionality of some indicators
func TestIndicatorValues(t *testing.T) {
	logger := zerolog.Nop()
	engine := NewStrategyEngine(logger)

	// Test data - simple ascending prices
	prices := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}
	high := []float64{10.5, 11.5, 12.5, 13.5, 14.5, 15.5, 16.5, 17.5, 18.5, 19.5, 20.5, 21.5, 22.5, 23.5, 24.5, 25.5}
	low := []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5, 15.5, 16.5, 17.5, 18.5, 19.5, 20.5, 21.5, 22.5, 23.5, 24.5}

	pricesList := engine.floatListToStarlark(prices)
	highList := engine.floatListToStarlark(high)
	lowList := engine.floatListToStarlark(low)

	// Test Heikin Ashi
	result, err := engine.heikinAshi(nil, nil, starlark.Tuple{pricesList, highList, lowList, pricesList}, nil)
	if err != nil {
		t.Fatalf("Error calling heikin_ashi: %v", err)
	}

	dict, ok := result.(*starlark.Dict)
	if !ok {
		t.Fatalf("heikin_ashi should return a dict")
	}

	// Check that all required keys exist
	requiredKeys := []string{"open", "high", "low", "close"}
	for _, key := range requiredKeys {
		if _, found, _ := dict.Get(starlark.String(key)); !found {
			t.Errorf("heikin_ashi result missing key: %s", key)
		}
	}

	t.Log("Heikin Ashi structure validation passed")

	// Test Supertrend
	result, err = engine.supertrend(nil, nil, starlark.Tuple{highList, lowList, pricesList, starlark.MakeInt(10), starlark.Float(3.0)}, nil)
	if err != nil {
		t.Fatalf("Error calling supertrend: %v", err)
	}

	dict, ok = result.(*starlark.Dict)
	if !ok {
		t.Fatalf("supertrend should return a dict")
	}

	// Check required keys
	requiredKeys = []string{"supertrend", "trend"}
	for _, key := range requiredKeys {
		if _, found, _ := dict.Get(starlark.String(key)); !found {
			t.Errorf("supertrend result missing key: %s", key)
		}
	}

	t.Log("Supertrend structure validation passed")
}
