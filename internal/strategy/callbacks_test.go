package strategy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestStartStopCallbacks(t *testing.T) {
	// Create a test strategy with on_start and on_stop callbacks
	testStrategy := `
# Test Strategy with Start/Stop Callbacks

def settings():
    return {
        "interval": "5m",
        "symbol": "BTCUSDT",
        "description": "Test strategy with start and stop callbacks"
    }

def on_start():
    print("Strategy started - initializing state")

def on_stop():
    print("Strategy stopped - cleaning up")

def on_kline(kline):
    current_price = float(kline.close)
    print("Processing kline: " + str(current_price))

def on_orderbook(orderbook):
    pass

def on_ticker(ticker):
    pass
`

	// Create strategy file in the strategy directory
	strategyDir := "strategy"
	if _, err := os.Stat(strategyDir); os.IsNotExist(err) {
		err = os.Mkdir(strategyDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
	}

	strategyPath := filepath.Join(strategyDir, "test_callbacks.star")
	err := os.WriteFile(strategyPath, []byte(testStrategy), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(strategyPath)

	// Create strategy engine
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	engine := NewStrategyEngine(logger)

	// Test ValidateCallbacks
	callbacks, err := engine.ValidateCallbacks("test_callbacks")
	if err != nil {
		t.Fatalf("Failed to validate callbacks: %v", err)
	}

	// Verify that start and stop callbacks are detected
	if !callbacks.HasOnStart {
		t.Error("Expected HasOnStart to be true")
	}
	if !callbacks.HasOnStop {
		t.Error("Expected HasOnStop to be true")
	}
	if !callbacks.HasOnKline {
		t.Error("Expected HasOnKline to be true")
	}

	t.Logf("Callback validation passed - HasOnStart: %v, HasOnStop: %v",
		callbacks.HasOnStart, callbacks.HasOnStop)

	// Test ExecuteStartCallback
	ctx := &StrategyContext{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Klines:    []*KlineData{},
		OrderBook: nil,
		Config:    map[string]interface{}{},
	}

	err = engine.ExecuteStartCallback("test_callbacks", ctx)
	if err != nil {
		t.Fatalf("ExecuteStartCallback failed: %v", err)
	}
	t.Log("ExecuteStartCallback executed successfully")

	// Test ExecuteStopCallback
	err = engine.ExecuteStopCallback("test_callbacks", ctx)
	if err != nil {
		t.Fatalf("ExecuteStopCallback failed: %v", err)
	}
	t.Log("ExecuteStopCallback executed successfully")
}

func TestCallbacksOptional(t *testing.T) {
	// Create a strategy without start/stop callbacks
	testStrategy := `
def settings():
    return {
        "interval": "1m",
        "symbol": "BTCUSDT"
    }

def on_kline(kline):
    pass
`

	// Create strategy file in the strategy directory
	strategyDir := "strategy"
	if _, err := os.Stat(strategyDir); os.IsNotExist(err) {
		err = os.Mkdir(strategyDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
	}

	strategyPath := filepath.Join(strategyDir, "test_minimal.star")
	err := os.WriteFile(strategyPath, []byte(testStrategy), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(strategyPath)

	// Create strategy engine
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	engine := NewStrategyEngine(logger)

	// Test ValidateCallbacks
	callbacks, err := engine.ValidateCallbacks("test_minimal")
	if err != nil {
		t.Fatalf("Failed to validate callbacks: %v", err)
	}

	// Verify that start and stop callbacks are NOT detected
	if callbacks.HasOnStart {
		t.Error("Expected HasOnStart to be false")
	}
	if callbacks.HasOnStop {
		t.Error("Expected HasOnStop to be false")
	}
	if !callbacks.HasOnKline {
		t.Error("Expected HasOnKline to be true")
	}

	t.Log("Optional callback validation passed - start/stop callbacks correctly detected as missing")

	// Test that calling start/stop callbacks on strategies without them doesn't fail
	ctx := &StrategyContext{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Klines:    []*KlineData{},
		OrderBook: nil,
		Config:    map[string]interface{}{},
	}

	// These should not fail even when callbacks don't exist
	err = engine.ExecuteStartCallback("test_minimal", ctx)
	if err != nil {
		t.Fatalf("ExecuteStartCallback should not fail for strategy without on_start: %v", err)
	}

	err = engine.ExecuteStopCallback("test_minimal", ctx)
	if err != nil {
		t.Fatalf("ExecuteStopCallback should not fail for strategy without on_stop: %v", err)
	}

	t.Log("Callback execution on minimal strategy succeeded")
}

func TestCallbackErrorHandling(t *testing.T) {
	// Create a strategy with failing callbacks
	testStrategy := `
def settings():
    return {"interval": "1m", "symbol": "BTCUSDT"}

def on_start():
    # This will cause an error
    undefined_variable + 1

def on_stop():
    # This will also cause an error
    another_undefined_variable.method()

def on_kline(kline):
    pass
`

	// Create strategy file in the strategy directory
	strategyDir := "strategy"
	if _, err := os.Stat(strategyDir); os.IsNotExist(err) {
		err = os.Mkdir(strategyDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
	}

	strategyPath := filepath.Join(strategyDir, "test_error.star")
	err := os.WriteFile(strategyPath, []byte(testStrategy), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(strategyPath)

	// Create strategy engine
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	engine := NewStrategyEngine(logger)

	ctx := &StrategyContext{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Klines:    []*KlineData{},
		OrderBook: nil,
		Config:    map[string]interface{}{},
	}

	// Test that errors in start callback are properly returned
	err = engine.ExecuteStartCallback("test_error", ctx)
	if err == nil {
		t.Error("Expected ExecuteStartCallback to return an error for failing on_start")
	}
	t.Logf("ExecuteStartCallback properly returned error: %v", err)

	// Test that errors in stop callback are properly returned
	err = engine.ExecuteStopCallback("test_error", ctx)
	if err == nil {
		t.Error("Expected ExecuteStopCallback to return an error for failing on_stop")
	}
	t.Logf("ExecuteStopCallback properly returned error: %v", err)
}
