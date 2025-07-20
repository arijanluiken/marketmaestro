package order

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
	"github.com/arijanluiken/mercantile/pkg/exchanges"
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

func TestConstants(t *testing.T) {
	// Test order type constants
	orderTypes := []string{
		OrderTypeMarket,
		OrderTypeLimit,
		OrderTypeStopMarket,
		OrderTypeStopLimit,
		OrderTypeTrailing,
	}

	expectedOrderTypes := []string{
		"market",
		"limit",
		"stop_market",
		"stop_limit",
		"trailing_stop",
	}

	for i, orderType := range orderTypes {
		if orderType != expectedOrderTypes[i] {
			t.Errorf("expected order type %s, got %s", expectedOrderTypes[i], orderType)
		}
	}

	// Test status constants
	statuses := []string{
		StatusPending,
		StatusOpen,
		StatusPartiallyFilled,
		StatusFilled,
		StatusCancelled,
		StatusRejected,
	}

	expectedStatuses := []string{
		"pending",
		"open",
		"partially_filled",
		"filled",
		"cancelled",
		"rejected",
	}

	for i, status := range statuses {
		if status != expectedStatuses[i] {
			t.Errorf("expected status %s, got %s", expectedStatuses[i], status)
		}
	}
}

func TestPlaceOrderMsg(t *testing.T) {
	msg := PlaceOrderMsg{
		Symbol:       "BTCUSDT",
		Side:         "buy",
		Type:         OrderTypeLimit,
		Quantity:     1.0,
		Price:        50000.0,
		StopPrice:    49000.0,
		TrailAmount:  100.0,
		TrailPercent: 1.0,
		TimeInForce:  "GTC",
		Reason:       "test order",
	}

	if msg.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", msg.Symbol)
	}
	if msg.Side != "buy" {
		t.Errorf("expected side buy, got %s", msg.Side)
	}
	if msg.Type != OrderTypeLimit {
		t.Errorf("expected type %s, got %s", OrderTypeLimit, msg.Type)
	}
	if msg.Quantity != 1.0 {
		t.Errorf("expected quantity 1.0, got %f", msg.Quantity)
	}
	if msg.Price != 50000.0 {
		t.Errorf("expected price 50000.0, got %f", msg.Price)
	}
	if msg.StopPrice != 49000.0 {
		t.Errorf("expected stop price 49000.0, got %f", msg.StopPrice)
	}
	if msg.TrailAmount != 100.0 {
		t.Errorf("expected trail amount 100.0, got %f", msg.TrailAmount)
	}
	if msg.TrailPercent != 1.0 {
		t.Errorf("expected trail percent 1.0, got %f", msg.TrailPercent)
	}
	if msg.TimeInForce != "GTC" {
		t.Errorf("expected time in force GTC, got %s", msg.TimeInForce)
	}
	if msg.Reason != "test order" {
		t.Errorf("expected reason 'test order', got %s", msg.Reason)
	}
}

func TestPlaceTrailingStopMsg(t *testing.T) {
	msg := PlaceTrailingStopMsg{
		Symbol:       "ETHUSDT",
		Side:         "sell",
		Quantity:     2.0,
		TrailAmount:  50.0,
		TrailPercent: 2.0,
		Reason:       "test trailing stop",
	}

	if msg.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", msg.Symbol)
	}
	if msg.Side != "sell" {
		t.Errorf("expected side sell, got %s", msg.Side)
	}
	if msg.Quantity != 2.0 {
		t.Errorf("expected quantity 2.0, got %f", msg.Quantity)
	}
	if msg.TrailAmount != 50.0 {
		t.Errorf("expected trail amount 50.0, got %f", msg.TrailAmount)
	}
	if msg.TrailPercent != 2.0 {
		t.Errorf("expected trail percent 2.0, got %f", msg.TrailPercent)
	}
	if msg.Reason != "test trailing stop" {
		t.Errorf("expected reason 'test trailing stop', got %s", msg.Reason)
	}
}

func TestPlaceStopOrderMsg(t *testing.T) {
	msg := PlaceStopOrderMsg{
		Symbol:     "ADAUSDT",
		Side:       "buy",
		Quantity:   100.0,
		StopPrice:  0.95,
		LimitPrice: 1.0,
		Reason:     "test stop order",
	}

	if msg.Symbol != "ADAUSDT" {
		t.Errorf("expected symbol ADAUSDT, got %s", msg.Symbol)
	}
	if msg.Side != "buy" {
		t.Errorf("expected side buy, got %s", msg.Side)
	}
	if msg.Quantity != 100.0 {
		t.Errorf("expected quantity 100.0, got %f", msg.Quantity)
	}
	if msg.StopPrice != 0.95 {
		t.Errorf("expected stop price 0.95, got %f", msg.StopPrice)
	}
	if msg.LimitPrice != 1.0 {
		t.Errorf("expected limit price 1.0, got %f", msg.LimitPrice)
	}
	if msg.Reason != "test stop order" {
		t.Errorf("expected reason 'test stop order', got %s", msg.Reason)
	}
}

func TestEnhancedOrder(t *testing.T) {
	now := time.Now()
	exchangeOrder := &exchanges.Order{
		ID:       "order123",
		Symbol:   "BTCUSDT",
		Side:     "buy",
		Type:     "limit",
		Quantity: 1.0,
		Price:    50000.0,
		Status:   "open",
		Time:     now,
	}

	enhancedOrder := &EnhancedOrder{
		Order:         exchangeOrder,
		OriginalType:  OrderTypeStopLimit,
		StopPrice:     49000.0,
		TrailAmount:   100.0,
		TrailPercent:  2.0,
		HighWaterMark: 51000.0,
		TimeInForce:   "GTC",
		CreatedAt:     now,
		UpdatedAt:     now.Add(time.Minute),
		TriggerPrice:  49500.0,
		IsTriggered:   true,
		ParentOrderID: "parent789",
	}

	if enhancedOrder.Order != exchangeOrder {
		t.Error("expected order to be set")
	}
	if enhancedOrder.OriginalType != OrderTypeStopLimit {
		t.Errorf("expected original type %s, got %s", OrderTypeStopLimit, enhancedOrder.OriginalType)
	}
	if enhancedOrder.StopPrice != 49000.0 {
		t.Errorf("expected stop price 49000.0, got %f", enhancedOrder.StopPrice)
	}
	if enhancedOrder.TrailAmount != 100.0 {
		t.Errorf("expected trail amount 100.0, got %f", enhancedOrder.TrailAmount)
	}
	if enhancedOrder.TrailPercent != 2.0 {
		t.Errorf("expected trail percent 2.0, got %f", enhancedOrder.TrailPercent)
	}
	if enhancedOrder.HighWaterMark != 51000.0 {
		t.Errorf("expected high water mark 51000.0, got %f", enhancedOrder.HighWaterMark)
	}
	if enhancedOrder.TimeInForce != "GTC" {
		t.Errorf("expected time in force GTC, got %s", enhancedOrder.TimeInForce)
	}
	if !enhancedOrder.CreatedAt.Equal(now) {
		t.Errorf("expected created at %v, got %v", now, enhancedOrder.CreatedAt)
	}
	if !enhancedOrder.UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Errorf("expected updated at %v, got %v", now.Add(time.Minute), enhancedOrder.UpdatedAt)
	}
	if enhancedOrder.TriggerPrice != 49500.0 {
		t.Errorf("expected trigger price 49500.0, got %f", enhancedOrder.TriggerPrice)
	}
	if !enhancedOrder.IsTriggered {
		t.Error("expected is triggered to be true")
	}
	if enhancedOrder.ParentOrderID != "parent789" {
		t.Errorf("expected parent order ID parent789, got %s", enhancedOrder.ParentOrderID)
	}
}

func TestNew(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDatabase(t)
	defer db.Close()
	logger := zerolog.New(nil)

	orderManager := New("bybit", cfg, db, logger)

	if orderManager == nil {
		t.Error("expected non-nil order manager")
	}
	if orderManager.exchangeName != "bybit" {
		t.Errorf("expected exchange name 'bybit', got '%s'", orderManager.exchangeName)
	}
	if orderManager.config != cfg {
		t.Error("expected config to be set")
	}
	if orderManager.db != db {
		t.Error("expected database to be set")
	}
	if orderManager.orders == nil {
		t.Error("expected orders map to be initialized")
	}
	if orderManager.stopOrders == nil {
		t.Error("expected stop orders map to be initialized")
	}
	if orderManager.trailingStops == nil {
		t.Error("expected trailing stops map to be initialized")
	}
	if orderManager.priceCache == nil {
		t.Error("expected price cache map to be initialized")
	}
	if orderManager.monitoringDone == nil {
		t.Error("expected monitoring done channel to be initialized")
	}
}

func TestSetExchange(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDatabase(t)
	defer db.Close()
	logger := zerolog.New(nil)

	orderManager := New("bybit", cfg, db, logger)
	
	// Create a mock exchange
	mockExchange := &mockExchange{name: "test_exchange"}
	
	orderManager.SetExchange(mockExchange)
	
	if orderManager.exchange == nil {
		t.Error("expected exchange to be set")
	}
	if orderManager.exchange.GetName() != "test_exchange" {
		t.Errorf("expected exchange name 'test_exchange', got '%s'", orderManager.exchange.GetName())
	}
}

func TestCancelOrderMsg(t *testing.T) {
	msg := CancelOrderMsg{
		OrderID: "order123",
		Symbol:  "BTCUSDT",
	}

	if msg.OrderID != "order123" {
		t.Errorf("expected order ID order123, got %s", msg.OrderID)
	}
	if msg.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", msg.Symbol)
	}
}

func TestOrderUpdateMsg(t *testing.T) {
	exchangeOrder := &exchanges.Order{
		ID:     "order123",
		Status: StatusFilled,
	}

	enhancedOrder := &EnhancedOrder{
		Order: exchangeOrder,
	}

	msg := OrderUpdateMsg{
		Order: enhancedOrder,
	}

	if msg.Order != enhancedOrder {
		t.Error("expected order to be set")
	}
	if msg.Order.Order.ID != "order123" {
		t.Errorf("expected order ID order123, got %s", msg.Order.Order.ID)
	}
	if msg.Order.Order.Status != StatusFilled {
		t.Errorf("expected status %s, got %s", StatusFilled, msg.Order.Order.Status)
	}
}

func TestPriceUpdateMsg(t *testing.T) {
	msg := PriceUpdateMsg{
		Symbol: "ETHUSDT",
		Price:  3000.0,
	}

	if msg.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", msg.Symbol)
	}
	if msg.Price != 3000.0 {
		t.Errorf("expected price 3000.0, got %f", msg.Price)
	}
}

// mockExchange is a simple mock implementation of exchanges.Exchange for testing
type mockExchange struct {
	name string
}

func (m *mockExchange) GetName() string { return m.name }
func (m *mockExchange) Connect(ctx context.Context) error { return nil }
func (m *mockExchange) Disconnect() error { return nil }
func (m *mockExchange) IsConnected() bool { return true }
func (m *mockExchange) SubscribeKlines(ctx context.Context, symbols []string, interval string, handler exchanges.DataHandler) error { return nil }
func (m *mockExchange) SubscribeOrderBook(ctx context.Context, symbols []string, handler exchanges.DataHandler) error { return nil }
func (m *mockExchange) UnsubscribeKlines(symbols []string) error { return nil }
func (m *mockExchange) UnsubscribeOrderBook(symbols []string) error { return nil }
func (m *mockExchange) PlaceOrder(ctx context.Context, order *exchanges.Order) (*exchanges.Order, error) { return order, nil }
func (m *mockExchange) CancelOrder(ctx context.Context, symbol, orderID string) error { return nil }
func (m *mockExchange) GetOrder(ctx context.Context, symbol, orderID string) (*exchanges.Order, error) { return nil, nil }
func (m *mockExchange) GetOpenOrders(ctx context.Context, symbol string) ([]*exchanges.Order, error) { return nil, nil }
func (m *mockExchange) GetBalances(ctx context.Context) ([]*exchanges.Balance, error) { return nil, nil }
func (m *mockExchange) GetPositions(ctx context.Context) ([]*exchanges.Position, error) { return nil, nil }
func (m *mockExchange) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]*exchanges.Kline, error) { return nil, nil }
func (m *mockExchange) GetOrderBook(ctx context.Context, symbol string, limit int) (*exchanges.OrderBook, error) { return nil, nil }
func (m *mockExchange) GetTicker(ctx context.Context, symbol string) (*exchanges.Ticker, error) { return nil, nil }
func (m *mockExchange) GetExchangeInfo(ctx context.Context) (*exchanges.ExchangeInfo, error) { return nil, nil }