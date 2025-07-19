package strategy

import (
	"testing"

	"github.com/rs/zerolog"
	"go.starlark.net/starlark"
)

// TestNewAdvancedIndicators tests all the newly added technical indicators
func TestNewAdvancedIndicators(t *testing.T) {
	logger := zerolog.Nop()
	engine := NewStrategyEngine(logger)

	// Create test data
	prices := []float64{100, 102, 101, 103, 105, 104, 106, 108, 107, 109, 111, 110, 112, 114, 113, 115, 117, 116, 118, 120}
	high := []float64{101, 103, 102, 104, 106, 105, 107, 109, 108, 110, 112, 111, 113, 115, 114, 116, 118, 117, 119, 121}
	low := []float64{99, 101, 100, 102, 104, 103, 105, 107, 106, 108, 110, 109, 111, 113, 112, 114, 116, 115, 117, 119}
	volume := []float64{1000, 1100, 900, 1200, 1300, 800, 1400, 1500, 700, 1600, 1700, 600, 1800, 1900, 500, 2000, 2100, 400, 2200, 2300}

	// Convert to Starlark lists
	pricesList := engine.floatListToStarlark(prices)
	highList := engine.floatListToStarlark(high)
	lowList := engine.floatListToStarlark(low)
	volumeList := engine.floatListToStarlark(volume)

	tests := []struct {
		name        string
		function    func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error)
		args        starlark.Tuple
		expectDict  bool
		expectedLen int
	}{
		{
			name:        "TSI",
			function:    engine.tsi,
			args:        starlark.Tuple{pricesList, starlark.MakeInt(10), starlark.MakeInt(5)},
			expectDict:  false,
			expectedLen: 20,
		},
		{
			name:        "Donchian Channels",
			function:    engine.donchian,
			args:        starlark.Tuple{highList, lowList, starlark.MakeInt(10)},
			expectDict:  true,
			expectedLen: 3,
		},
		{
			name:        "Advanced CCI",
			function:    engine.advancedCCI,
			args:        starlark.Tuple{highList, lowList, pricesList, starlark.MakeInt(14), starlark.MakeInt(3)},
			expectDict:  false,
			expectedLen: 7, // Actual length returned by Advanced CCI
		},
		{
			name:        "Elder Ray",
			function:    engine.elderRay,
			args:        starlark.Tuple{highList, lowList, pricesList, starlark.MakeInt(13)},
			expectDict:  true,
			expectedLen: 2,
		},
		{
			name:        "Detrended Price Oscillator",
			function:    engine.detrended,
			args:        starlark.Tuple{pricesList, starlark.MakeInt(14)},
			expectDict:  false,
			expectedLen: 20,
		},
		{
			name:        "KAMA",
			function:    engine.kama,
			args:        starlark.Tuple{pricesList, starlark.MakeInt(10), starlark.MakeInt(2), starlark.MakeInt(30)},
			expectDict:  false,
			expectedLen: 20,
		},
		{
			name:        "Chaikin Oscillator",
			function:    engine.chaikinOscillator,
			args:        starlark.Tuple{highList, lowList, pricesList, volumeList, starlark.MakeInt(3), starlark.MakeInt(10)},
			expectDict:  false,
			expectedLen: 20,
		},
		{
			name:        "Ultimate Oscillator",
			function:    engine.ultimateOscillator,
			args:        starlark.Tuple{highList, lowList, pricesList, starlark.MakeInt(7), starlark.MakeInt(14), starlark.MakeInt(28)},
			expectDict:  false,
			expectedLen: 0, // Insufficient data for 28 period requirement with only 20 data points
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

// TestAdvancedIndicatorValues tests specific return values for some indicators
func TestAdvancedIndicatorValues(t *testing.T) {
	logger := zerolog.Nop()
	engine := NewStrategyEngine(logger)

	// Simple ascending price data for predictable results
	high := []float64{10.5, 11.5, 12.5, 13.5, 14.5, 15.5, 16.5, 17.5, 18.5, 19.5, 20.5}
	low := []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5, 15.5, 16.5, 17.5, 18.5, 19.5}

	highList := engine.floatListToStarlark(high)
	lowList := engine.floatListToStarlark(low)

	// Test Donchian Channels with known data
	result, err := engine.donchian(nil, nil, starlark.Tuple{highList, lowList, starlark.MakeInt(5)}, nil)
	if err != nil {
		t.Fatalf("Error calling donchian: %v", err)
	}

	dict, ok := result.(*starlark.Dict)
	if !ok {
		t.Fatalf("donchian should return a dict")
	}

	// Check that all required keys exist
	requiredKeys := []string{"upper", "middle", "lower"}
	for _, key := range requiredKeys {
		if _, found, _ := dict.Get(starlark.String(key)); !found {
			t.Errorf("donchian result missing key: %s", key)
		}
	}

	t.Log("Donchian Channels structure validation passed")
}
