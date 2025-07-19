package exchanges

import (
	"context"
	"time"
)

// Kline represents a candlestick/kline data point
type Kline struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// OrderBookEntry represents a single order book entry
type OrderBookEntry struct {
	Price    float64
	Quantity float64
}

// OrderBook represents the order book data
type OrderBook struct {
	Symbol    string
	Timestamp time.Time
	Bids      []OrderBookEntry
	Asks      []OrderBookEntry
}

// Order represents a trading order
type Order struct {
	ID       string
	Symbol   string
	Side     string // "buy" or "sell"
	Type     string // "market", "limit", etc.
	Quantity float64
	Price    float64
	Status   string
	Time     time.Time
}

// Position represents a trading position
type Position struct {
	Symbol       string
	Side         string // "long" or "short"
	Size         float64
	EntryPrice   float64
	MarkPrice    float64
	UnrealizedPL float64
	Timestamp    time.Time
}

// Balance represents account balance
type Balance struct {
	Asset     string
	Available float64
	Locked    float64
	Total     float64
}

// DataHandler defines callbacks for receiving data
type DataHandler interface {
	OnKline(kline *Kline)
	OnOrderBook(orderbook *OrderBook)
	OnOrder(order *Order)
	OnPosition(position *Position)
	OnBalance(balance *Balance)
}

// Exchange defines the interface for cryptocurrency exchanges
type Exchange interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	// Data subscriptions
	SubscribeKlines(ctx context.Context, symbols []string, interval string, handler DataHandler) error
	SubscribeOrderBook(ctx context.Context, symbols []string, handler DataHandler) error
	UnsubscribeKlines(symbols []string) error
	UnsubscribeOrderBook(symbols []string) error

	// Trading operations
	PlaceOrder(ctx context.Context, order *Order) (*Order, error)
	CancelOrder(ctx context.Context, symbol, orderID string) error
	GetOrder(ctx context.Context, symbol, orderID string) (*Order, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)

	// Account information
	GetBalances(ctx context.Context) ([]*Balance, error)
	GetPositions(ctx context.Context) ([]*Position, error)

	// Market data
	GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]*Kline, error)
	GetOrderBook(ctx context.Context, symbol string, limit int) (*OrderBook, error)
	GetTicker(ctx context.Context, symbol string) (*Ticker, error)

	// Exchange information
	GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error)
	GetName() string
}

// Ticker represents ticker information
type Ticker struct {
	Symbol    string
	Price     float64
	Volume    float64
	Change    float64
	ChangeP   float64
	Timestamp time.Time
}

// Symbol represents trading pair information
type Symbol struct {
	Name              string
	BaseAsset         string
	QuoteAsset        string
	Status            string
	MinOrderSize      float64
	MaxOrderSize      float64
	MinPrice          float64
	MaxPrice          float64
	PricePrecision    int
	QuantityPrecision int
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	Name    string
	Symbols []*Symbol
}