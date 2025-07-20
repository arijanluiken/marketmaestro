package portfolio

import (
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

func setupTestPortfolio(t *testing.T) (*PortfolioActor, *database.DB) {
	db := setupTestDatabase(t)
	cfg := &config.Config{}
	logger := zerolog.New(nil)

	portfolio := New("test_exchange", cfg, db, logger)
	return portfolio, db
}

func TestNew(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDatabase(t)
	defer db.Close()
	logger := zerolog.New(nil)

	portfolio := New("bybit", cfg, db, logger)

	if portfolio == nil {
		t.Error("expected non-nil portfolio")
	}
	if portfolio.exchangeName != "bybit" {
		t.Errorf("expected exchange name 'bybit', got '%s'", portfolio.exchangeName)
	}
	if portfolio.config != cfg {
		t.Error("expected config to be set")
	}
	if portfolio.db != db {
		t.Error("expected database to be set")
	}
	if portfolio.positions == nil {
		t.Error("expected positions map to be initialized")
	}
	if portfolio.balances == nil {
		t.Error("expected balances map to be initialized")
	}
	if portfolio.trades == nil {
		t.Error("expected trades slice to be initialized")
	}
	if portfolio.pnlHistory == nil {
		t.Error("expected PnL history map to be initialized")
	}
	if portfolio.currentPrices == nil {
		t.Error("expected current prices map to be initialized")
	}
	if portfolio.syncInterval != 5*time.Minute {
		t.Errorf("expected sync interval 5 minutes, got %v", portfolio.syncInterval)
	}
}

func TestSetExchangeActor(t *testing.T) {
	portfolio, db := setupTestPortfolio(t)
	defer db.Close()

	// Test that function doesn't panic
	portfolio.SetExchangeActor(nil)

	if portfolio.exchangeActorPID != nil {
		t.Error("expected exchange actor PID to be nil when passing nil")
	}
}

func TestPosition(t *testing.T) {
	now := time.Now()
	position := Position{
		Exchange:      "bybit",
		Symbol:        "BTCUSDT",
		Quantity:      1.5,
		AveragePrice:  48000.0,
		CurrentPrice:  50000.0,
		UnrealizedPnL: 3000.0,
		UpdatedAt:     now,
	}

	if position.Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", position.Exchange)
	}
	if position.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", position.Symbol)
	}
	if position.Quantity != 1.5 {
		t.Errorf("expected quantity 1.5, got %f", position.Quantity)
	}
	if position.AveragePrice != 48000.0 {
		t.Errorf("expected average price 48000.0, got %f", position.AveragePrice)
	}
	if position.CurrentPrice != 50000.0 {
		t.Errorf("expected current price 50000.0, got %f", position.CurrentPrice)
	}
	if position.UnrealizedPnL != 3000.0 {
		t.Errorf("expected unrealized PnL 3000.0, got %f", position.UnrealizedPnL)
	}
	if !position.UpdatedAt.Equal(now) {
		t.Errorf("expected updated at %v, got %v", now, position.UpdatedAt)
	}
}

func TestBalance(t *testing.T) {
	now := time.Now()
	balance := Balance{
		Exchange:  "bitvavo",
		Asset:     "BTC",
		Available: 1.5,
		Locked:    0.5,
		Total:     2.0,
		UpdatedAt: now,
	}

	if balance.Exchange != "bitvavo" {
		t.Errorf("expected exchange bitvavo, got %s", balance.Exchange)
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
	if !balance.UpdatedAt.Equal(now) {
		t.Errorf("expected updated at %v, got %v", now, balance.UpdatedAt)
	}
}

func TestTrade(t *testing.T) {
	now := time.Now()
	trade := Trade{
		ID:        "trade123",
		Exchange:  "bybit",
		Symbol:    "ETHUSDT",
		Side:      "buy",
		Quantity:  2.0,
		Price:     3000.0,
		Fee:       5.0,
		Timestamp: now,
	}

	if trade.ID != "trade123" {
		t.Errorf("expected ID trade123, got %s", trade.ID)
	}
	if trade.Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", trade.Exchange)
	}
	if trade.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", trade.Symbol)
	}
	if trade.Side != "buy" {
		t.Errorf("expected side buy, got %s", trade.Side)
	}
	if trade.Quantity != 2.0 {
		t.Errorf("expected quantity 2.0, got %f", trade.Quantity)
	}
	if trade.Price != 3000.0 {
		t.Errorf("expected price 3000.0, got %f", trade.Price)
	}
	if trade.Fee != 5.0 {
		t.Errorf("expected fee 5.0, got %f", trade.Fee)
	}
	if !trade.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, trade.Timestamp)
	}
}

func TestUpdatePositionMsg(t *testing.T) {
	msg := UpdatePositionMsg{
		Exchange: "bybit",
		Symbol:   "BTCUSDT",
		Quantity: 1.0,
		Price:    50000.0,
		Side:     "buy",
	}

	if msg.Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", msg.Exchange)
	}
	if msg.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", msg.Symbol)
	}
	if msg.Quantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", msg.Quantity)
	}
	if msg.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", msg.Price)
	}
	if msg.Side != "buy" {
		t.Errorf("expected side buy, got %s", msg.Side)
	}
}

func TestUpdateBalanceMsg(t *testing.T) {
	msg := UpdateBalanceMsg{
		Exchange: "bitvavo",
		Asset:    "ETH",
		Amount:   10.0,
	}

	if msg.Exchange != "bitvavo" {
		t.Errorf("expected exchange bitvavo, got %s", msg.Exchange)
	}
	if msg.Asset != "ETH" {
		t.Errorf("expected asset ETH, got %s", msg.Asset)
	}
	if msg.Amount != 10.0 {
		t.Errorf("expected amount 10.0, got %f", msg.Amount)
	}
}

func TestTradeExecutedMsg(t *testing.T) {
	trade := Trade{
		ID:       "trade456",
		Exchange: "bybit",
		Symbol:   "ADAUSDT",
		Side:     "sell",
		Quantity: 100.0,
		Price:    1.0,
		Fee:      0.1,
	}

	msg := TradeExecutedMsg{
		Trade: trade,
	}

	if msg.Trade.ID != "trade456" {
		t.Errorf("expected trade ID trade456, got %s", msg.Trade.ID)
	}
	if msg.Trade.Exchange != "bybit" {
		t.Errorf("expected exchange bybit, got %s", msg.Trade.Exchange)
	}
	if msg.Trade.Symbol != "ADAUSDT" {
		t.Errorf("expected symbol ADAUSDT, got %s", msg.Trade.Symbol)
	}
	if msg.Trade.Side != "sell" {
		t.Errorf("expected side sell, got %s", msg.Trade.Side)
	}
}

func TestUpdateMarketPricesMsg(t *testing.T) {
	prices := map[string]float64{
		"BTCUSDT": 50000.0,
		"ETHUSDT": 3000.0,
		"ADAUSDT": 1.0,
	}

	msg := UpdateMarketPricesMsg{
		Prices: prices,
	}

	if len(msg.Prices) != 3 {
		t.Errorf("expected 3 prices, got %d", len(msg.Prices))
	}
	if msg.Prices["BTCUSDT"] != 50000.0 {
		t.Errorf("expected BTCUSDT price 50000.0, got %f", msg.Prices["BTCUSDT"])
	}
	if msg.Prices["ETHUSDT"] != 3000.0 {
		t.Errorf("expected ETHUSDT price 3000.0, got %f", msg.Prices["ETHUSDT"])
	}
	if msg.Prices["ADAUSDT"] != 1.0 {
		t.Errorf("expected ADAUSDT price 1.0, got %f", msg.Prices["ADAUSDT"])
	}
}

func TestPositionsResponse(t *testing.T) {
	positions := []Position{
		{
			Exchange: "bybit",
			Symbol:   "BTCUSDT",
			Quantity: 1.0,
		},
		{
			Exchange: "bybit",
			Symbol:   "ETHUSDT",
			Quantity: 2.0,
		},
	}

	response := PositionsResponse{
		Positions: positions,
	}

	if len(response.Positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(response.Positions))
	}
	if response.Positions[0].Symbol != "BTCUSDT" {
		t.Errorf("expected first position symbol BTCUSDT, got %s", response.Positions[0].Symbol)
	}
	if response.Positions[1].Symbol != "ETHUSDT" {
		t.Errorf("expected second position symbol ETHUSDT, got %s", response.Positions[1].Symbol)
	}
}

func TestBalancesResponse(t *testing.T) {
	balances := []Balance{
		{
			Exchange: "bybit",
			Asset:    "BTC",
			Total:    1.5,
		},
		{
			Exchange: "bybit",
			Asset:    "USDT",
			Total:    10000.0,
		},
	}

	response := BalancesResponse{
		Balances: balances,
	}

	if len(response.Balances) != 2 {
		t.Errorf("expected 2 balances, got %d", len(response.Balances))
	}
	if response.Balances[0].Asset != "BTC" {
		t.Errorf("expected first balance asset BTC, got %s", response.Balances[0].Asset)
	}
	if response.Balances[1].Asset != "USDT" {
		t.Errorf("expected second balance asset USDT, got %s", response.Balances[1].Asset)
	}
}

func TestPerformanceResponse(t *testing.T) {
	response := PerformanceResponse{
		TotalValue:    100000.0,
		AvailableCash: 50000.0,
		UnrealizedPnL: 5000.0,
		RealizedPnL:   2000.0,
		DailyPnL:      1000.0,
		WeeklyPnL:     3000.0,
		MonthlyPnL:    8000.0,
	}

	if response.TotalValue != 100000.0 {
		t.Errorf("expected total value 100000.0, got %f", response.TotalValue)
	}
	if response.AvailableCash != 50000.0 {
		t.Errorf("expected available cash 50000.0, got %f", response.AvailableCash)
	}
	if response.UnrealizedPnL != 5000.0 {
		t.Errorf("expected unrealized PnL 5000.0, got %f", response.UnrealizedPnL)
	}
	if response.RealizedPnL != 2000.0 {
		t.Errorf("expected realized PnL 2000.0, got %f", response.RealizedPnL)
	}
	if response.DailyPnL != 1000.0 {
		t.Errorf("expected daily PnL 1000.0, got %f", response.DailyPnL)
	}
	if response.WeeklyPnL != 3000.0 {
		t.Errorf("expected weekly PnL 3000.0, got %f", response.WeeklyPnL)
	}
	if response.MonthlyPnL != 8000.0 {
		t.Errorf("expected monthly PnL 8000.0, got %f", response.MonthlyPnL)
	}
}

func TestEmptyMessages(t *testing.T) {
	// Test that message structs can be created without panicking
	_ = SyncWithExchangeMsg{}
	_ = RequestBalancesMsg{}
	_ = RequestPositionsMsg{}
	_ = GetPositionsMsg{}
	_ = GetBalancesMsg{}
	_ = GetPerformanceMsg{}
	_ = StatusMsg{}
}

func TestSetExchangeActorMsg(t *testing.T) {
	msg := SetExchangeActorMsg{
		ExchangeActorPID: nil, // We can't create real PIDs in tests
	}

	if msg.ExchangeActorPID != nil {
		t.Error("expected exchange actor PID to be nil")
	}
}

func TestPortfolioDataAccess(t *testing.T) {
	portfolio, db := setupTestPortfolio(t)
	defer db.Close()

	// Test initial state
	if len(portfolio.positions) != 0 {
		t.Errorf("expected 0 initial positions, got %d", len(portfolio.positions))
	}
	if len(portfolio.balances) != 0 {
		t.Errorf("expected 0 initial balances, got %d", len(portfolio.balances))
	}
	if len(portfolio.trades) != 0 {
		t.Errorf("expected 0 initial trades, got %d", len(portfolio.trades))
	}
	if len(portfolio.pnlHistory) != 0 {
		t.Errorf("expected 0 initial PnL history entries, got %d", len(portfolio.pnlHistory))
	}
	if len(portfolio.currentPrices) != 0 {
		t.Errorf("expected 0 initial current prices, got %d", len(portfolio.currentPrices))
	}

	// Test that we can add data directly to the portfolio (for testing purposes)
	portfolio.positions["test_exchange:BTCUSDT"] = &Position{
		Exchange: "test_exchange",
		Symbol:   "BTCUSDT",
		Quantity: 1.0,
	}

	if len(portfolio.positions) != 1 {
		t.Errorf("expected 1 position after adding, got %d", len(portfolio.positions))
	}

	position := portfolio.positions["test_exchange:BTCUSDT"]
	if position.Symbol != "BTCUSDT" {
		t.Errorf("expected position symbol BTCUSDT, got %s", position.Symbol)
	}
}

func TestPortfolioCalculations(t *testing.T) {
	portfolio, db := setupTestPortfolio(t)
	defer db.Close()

	// Add test data
	portfolio.positions["test_exchange:BTCUSDT"] = &Position{
		Exchange:     "test_exchange",
		Symbol:       "BTCUSDT",
		Quantity:     1.0,
		AveragePrice: 48000.0,
		CurrentPrice: 50000.0,
	}

	portfolio.currentPrices["BTCUSDT"] = 50000.0

	// Test position value calculation (manually)
	position := portfolio.positions["test_exchange:BTCUSDT"]
	expectedUnrealizedPnL := (position.CurrentPrice - position.AveragePrice) * position.Quantity
	if expectedUnrealizedPnL != 2000.0 {
		t.Errorf("expected unrealized PnL 2000.0, got %f", expectedUnrealizedPnL)
	}

	// Test that position value is quantity * current price
	expectedPositionValue := position.Quantity * position.CurrentPrice
	if expectedPositionValue != 50000.0 {
		t.Errorf("expected position value 50000.0, got %f", expectedPositionValue)
	}
}