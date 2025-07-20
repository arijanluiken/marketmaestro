package exchanges

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestNewFactory(t *testing.T) {
	logger := log.With().Str("test", "factory").Logger()
	
	factory := NewFactory(logger)
	if factory == nil {
		t.Error("expected non-nil factory")
	}
}

func TestFactoryGetSupportedExchanges(t *testing.T) {
	logger := zerolog.New(nil)
	factory := NewFactory(logger)
	
	supported := factory.GetSupportedExchanges()
	expected := []string{"bybit", "bitvavo"}
	
	if len(supported) != len(expected) {
		t.Errorf("expected %d supported exchanges, got %d", len(expected), len(supported))
	}
	
	for i, exchange := range expected {
		if i >= len(supported) || supported[i] != exchange {
			t.Errorf("expected exchange %s at index %d, got %s", exchange, i, supported[i])
		}
	}
}

func TestFactoryCreateExchange(t *testing.T) {
	logger := zerolog.New(nil)
	factory := NewFactory(logger)

	t.Run("creates bybit exchange", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
			"secret":  "test_secret",
			"testnet": true,
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err != nil {
			t.Fatalf("expected no error creating bybit exchange, got %v", err)
		}

		if exchange == nil {
			t.Error("expected non-nil exchange")
		}

		if exchange.GetName() != "bybit" {
			t.Errorf("expected exchange name 'bybit', got '%s'", exchange.GetName())
		}
	})

	t.Run("creates bitvavo exchange", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
			"secret":  "test_secret",
			"testnet": false,
		}

		exchange, err := factory.CreateExchange("bitvavo", config)
		if err != nil {
			t.Fatalf("expected no error creating bitvavo exchange, got %v", err)
		}

		if exchange == nil {
			t.Error("expected non-nil exchange")
		}

		if exchange.GetName() != "bitvavo" {
			t.Errorf("expected exchange name 'bitvavo', got '%s'", exchange.GetName())
		}
	})

	t.Run("fails for unsupported exchange", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
			"secret":  "test_secret",
		}

		exchange, err := factory.CreateExchange("unsupported", config)
		if err == nil {
			t.Error("expected error for unsupported exchange, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for unsupported exchange")
		}

		expectedMsg := "unsupported exchange: unsupported"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("fails when bybit api_key missing", func(t *testing.T) {
		config := map[string]interface{}{
			"secret": "test_secret",
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err == nil {
			t.Error("expected error for missing bybit api_key, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for missing api_key")
		}

		expectedMsg := "bybit api_key is required"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("fails when bybit api_key empty", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "",
			"secret":  "test_secret",
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err == nil {
			t.Error("expected error for empty bybit api_key, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for empty api_key")
		}
	})

	t.Run("fails when bybit secret missing", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err == nil {
			t.Error("expected error for missing bybit secret, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for missing secret")
		}

		expectedMsg := "bybit secret is required"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("fails when bybit secret empty", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
			"secret":  "",
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err == nil {
			t.Error("expected error for empty bybit secret, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for empty secret")
		}
	})

	t.Run("fails when bitvavo api_key missing", func(t *testing.T) {
		config := map[string]interface{}{
			"secret": "test_secret",
		}

		exchange, err := factory.CreateExchange("bitvavo", config)
		if err == nil {
			t.Error("expected error for missing bitvavo api_key, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for missing api_key")
		}

		expectedMsg := "bitvavo api_key is required"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("fails when bitvavo secret missing", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
		}

		exchange, err := factory.CreateExchange("bitvavo", config)
		if err == nil {
			t.Error("expected error for missing bitvavo secret, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for missing secret")
		}

		expectedMsg := "bitvavo secret is required"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("handles wrong type for api_key", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": 123, // wrong type
			"secret":  "test_secret",
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err == nil {
			t.Error("expected error for wrong api_key type, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for wrong api_key type")
		}
	})

	t.Run("handles wrong type for secret", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
			"secret":  123, // wrong type
		}

		exchange, err := factory.CreateExchange("bybit", config)
		if err == nil {
			t.Error("expected error for wrong secret type, got nil")
		}
		if exchange != nil {
			t.Error("expected nil exchange for wrong secret type")
		}
	})

	t.Run("handles testnet flag correctly", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key": "test_key",
			"secret":  "test_secret",
			"testnet": "not_a_bool", // wrong type, should be ignored
		}

		// Should not fail - testnet is optional and defaults to false
		exchange, err := factory.CreateExchange("bybit", config)
		if err != nil {
			t.Fatalf("expected no error with wrong testnet type, got %v", err)
		}
		if exchange == nil {
			t.Error("expected non-nil exchange")
		}
	})
}

func TestKlineStruct(t *testing.T) {
	now := time.Now()
	kline := &Kline{
		Symbol:    "BTCUSDT",
		Timestamp: now,
		Open:      50000.0,
		High:      51000.0,
		Low:       49000.0,
		Close:     50500.0,
		Volume:    100.0,
		Interval:  "1m",
	}

	if kline.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", kline.Symbol)
	}
	if !kline.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, kline.Timestamp)
	}
	if kline.Open != 50000.0 {
		t.Errorf("expected open 50000.0, got %f", kline.Open)
	}
	if kline.High != 51000.0 {
		t.Errorf("expected high 51000.0, got %f", kline.High)
	}
	if kline.Low != 49000.0 {
		t.Errorf("expected low 49000.0, got %f", kline.Low)
	}
	if kline.Close != 50500.0 {
		t.Errorf("expected close 50500.0, got %f", kline.Close)
	}
	if kline.Volume != 100.0 {
		t.Errorf("expected volume 100.0, got %f", kline.Volume)
	}
	if kline.Interval != "1m" {
		t.Errorf("expected interval 1m, got %s", kline.Interval)
	}
}

func TestOrderBookEntryStruct(t *testing.T) {
	entry := &OrderBookEntry{
		Price:    50000.0,
		Quantity: 1.5,
	}

	if entry.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", entry.Price)
	}
	if entry.Quantity != 1.5 {
		t.Errorf("expected quantity 1.5, got %f", entry.Quantity)
	}
}

func TestOrderBookStruct(t *testing.T) {
	now := time.Now()
	orderBook := &OrderBook{
		Symbol:    "ETHUSDT",
		Timestamp: now,
		Bids: []OrderBookEntry{
			{Price: 3000.0, Quantity: 1.0},
			{Price: 2999.0, Quantity: 2.0},
		},
		Asks: []OrderBookEntry{
			{Price: 3001.0, Quantity: 1.5},
			{Price: 3002.0, Quantity: 0.5},
		},
	}

	if orderBook.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", orderBook.Symbol)
	}
	if !orderBook.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, orderBook.Timestamp)
	}
	if len(orderBook.Bids) != 2 {
		t.Errorf("expected 2 bids, got %d", len(orderBook.Bids))
	}
	if len(orderBook.Asks) != 2 {
		t.Errorf("expected 2 asks, got %d", len(orderBook.Asks))
	}
	if orderBook.Bids[0].Price != 3000.0 {
		t.Errorf("expected first bid price 3000.0, got %f", orderBook.Bids[0].Price)
	}
	if orderBook.Asks[0].Quantity != 1.5 {
		t.Errorf("expected first ask quantity 1.5, got %f", orderBook.Asks[0].Quantity)
	}
}

func TestOrderStruct(t *testing.T) {
	now := time.Now()
	order := &Order{
		ID:       "12345",
		Symbol:   "BTCUSDT",
		Side:     "buy",
		Type:     "limit",
		Quantity: 1.0,
		Price:    50000.0,
		Status:   "open",
		Time:     now,
	}

	if order.ID != "12345" {
		t.Errorf("expected ID 12345, got %s", order.ID)
	}
	if order.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", order.Symbol)
	}
	if order.Side != "buy" {
		t.Errorf("expected side buy, got %s", order.Side)
	}
	if order.Type != "limit" {
		t.Errorf("expected type limit, got %s", order.Type)
	}
	if order.Quantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", order.Quantity)
	}
	if order.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", order.Price)
	}
	if order.Status != "open" {
		t.Errorf("expected status open, got %s", order.Status)
	}
	if !order.Time.Equal(now) {
		t.Errorf("expected time %v, got %v", now, order.Time)
	}
}

func TestPositionStruct(t *testing.T) {
	now := time.Now()
	position := &Position{
		Symbol:       "BTCUSDT",
		Side:         "long",
		Size:         1.5,
		EntryPrice:   49000.0,
		MarkPrice:    50000.0,
		UnrealizedPL: 1500.0,
		Timestamp:    now,
	}

	if position.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", position.Symbol)
	}
	if position.Side != "long" {
		t.Errorf("expected side long, got %s", position.Side)
	}
	if position.Size != 1.5 {
		t.Errorf("expected size 1.5, got %f", position.Size)
	}
	if position.EntryPrice != 49000.0 {
		t.Errorf("expected entry price 49000.0, got %f", position.EntryPrice)
	}
	if position.MarkPrice != 50000.0 {
		t.Errorf("expected mark price 50000.0, got %f", position.MarkPrice)
	}
	if position.UnrealizedPL != 1500.0 {
		t.Errorf("expected unrealized PL 1500.0, got %f", position.UnrealizedPL)
	}
	if !position.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, position.Timestamp)
	}
}

func TestBalanceStruct(t *testing.T) {
	balance := &Balance{
		Asset:     "BTC",
		Available: 1.5,
		Locked:    0.5,
		Total:     2.0,
	}

	if balance.Asset != "BTC" {
		t.Errorf("expected asset BTC, got %s", balance.Asset)
	}
	if balance.Available != 1.5 {
		t.Errorf("expected available 1.5, got %f", balance.Available)
	}
	if balance.Locked != 0.5 {
		t.Errorf("expected locked 0.5, got %f", balance.Locked)
	}
	if balance.Total != 2.0 {
		t.Errorf("expected total 2.0, got %f", balance.Total)
	}
}

func TestTickerStruct(t *testing.T) {
	now := time.Now()
	ticker := &Ticker{
		Symbol:    "BTCUSDT",
		Price:     50000.0,
		Volume:    1000.0,
		Change:    500.0,
		ChangeP:   1.01,
		Timestamp: now,
	}

	if ticker.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", ticker.Symbol)
	}
	if ticker.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", ticker.Price)
	}
	if ticker.Volume != 1000.0 {
		t.Errorf("expected volume 1000.0, got %f", ticker.Volume)
	}
	if ticker.Change != 500.0 {
		t.Errorf("expected change 500.0, got %f", ticker.Change)
	}
	if ticker.ChangeP != 1.01 {
		t.Errorf("expected change percent 1.01, got %f", ticker.ChangeP)
	}
	if !ticker.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, ticker.Timestamp)
	}
}

func TestSymbolStruct(t *testing.T) {
	symbol := &Symbol{
		Name:              "BTCUSDT",
		BaseAsset:         "BTC",
		QuoteAsset:        "USDT",
		Status:            "active",
		MinOrderSize:      0.001,
		MaxOrderSize:      1000.0,
		MinPrice:          0.01,
		MaxPrice:          100000.0,
		PricePrecision:    2,
		QuantityPrecision: 8,
	}

	if symbol.Name != "BTCUSDT" {
		t.Errorf("expected name BTCUSDT, got %s", symbol.Name)
	}
	if symbol.BaseAsset != "BTC" {
		t.Errorf("expected base asset BTC, got %s", symbol.BaseAsset)
	}
	if symbol.QuoteAsset != "USDT" {
		t.Errorf("expected quote asset USDT, got %s", symbol.QuoteAsset)
	}
	if symbol.Status != "active" {
		t.Errorf("expected status active, got %s", symbol.Status)
	}
	if symbol.MinOrderSize != 0.001 {
		t.Errorf("expected min order size 0.001, got %f", symbol.MinOrderSize)
	}
	if symbol.MaxOrderSize != 1000.0 {
		t.Errorf("expected max order size 1000.0, got %f", symbol.MaxOrderSize)
	}
	if symbol.MinPrice != 0.01 {
		t.Errorf("expected min price 0.01, got %f", symbol.MinPrice)
	}
	if symbol.MaxPrice != 100000.0 {
		t.Errorf("expected max price 100000.0, got %f", symbol.MaxPrice)
	}
	if symbol.PricePrecision != 2 {
		t.Errorf("expected price precision 2, got %d", symbol.PricePrecision)
	}
	if symbol.QuantityPrecision != 8 {
		t.Errorf("expected quantity precision 8, got %d", symbol.QuantityPrecision)
	}
}

func TestExchangeInfoStruct(t *testing.T) {
	symbols := []*Symbol{
		{Name: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT"},
		{Name: "ETHUSDT", BaseAsset: "ETH", QuoteAsset: "USDT"},
	}

	exchangeInfo := &ExchangeInfo{
		Name:    "test_exchange",
		Symbols: symbols,
	}

	if exchangeInfo.Name != "test_exchange" {
		t.Errorf("expected name test_exchange, got %s", exchangeInfo.Name)
	}
	if len(exchangeInfo.Symbols) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(exchangeInfo.Symbols))
	}
	if exchangeInfo.Symbols[0].Name != "BTCUSDT" {
		t.Errorf("expected first symbol BTCUSDT, got %s", exchangeInfo.Symbols[0].Name)
	}
}

func TestBitvavoExchange(t *testing.T) {
	logger := zerolog.New(nil)

	t.Run("creates new bitvavo exchange", func(t *testing.T) {
		exchange := NewBitvavo("test_key", "test_secret", true, logger)
		
		if exchange == nil {
			t.Error("expected non-nil exchange")
		}
		if exchange.GetName() != "bitvavo" {
			t.Errorf("expected name bitvavo, got %s", exchange.GetName())
		}
		if !exchange.testnet {
			t.Error("expected testnet to be true")
		}
	})

	t.Run("connects to bitvavo", func(t *testing.T) {
		exchange := NewBitvavo("test_key", "test_secret", false, logger)
		
		ctx := context.Background()
		err := exchange.Connect(ctx)
		if err != nil {
			t.Errorf("expected no error connecting, got %v", err)
		}
	})

	t.Run("disconnects from bitvavo", func(t *testing.T) {
		exchange := NewBitvavo("test_key", "test_secret", false, logger)
		
		err := exchange.Disconnect()
		if err != nil {
			t.Errorf("expected no error disconnecting, got %v", err)
		}
	})

	t.Run("reports connected when client exists", func(t *testing.T) {
		exchange := NewBitvavo("test_key", "test_secret", false, logger)
		
		// Client is initialized in NewBitvavo, so IsConnected should return true
		if !exchange.IsConnected() {
			t.Error("expected IsConnected to return true when client is initialized")
		}
	})
}