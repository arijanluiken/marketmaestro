package exchanges

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hirokisan/bybit/v2"
	"github.com/rs/zerolog"
)

// BybitExchange implements the Exchange interface for Bybit
type BybitExchange struct {
	client  *bybit.Client
	logger  zerolog.Logger
	name    string
	testnet bool

	// WebSocket connections
	wsConn    *websocket.Conn
	wsConnMu  sync.RWMutex
	connected bool

	// Subscription management
	subscriptions map[string]DataHandler
	subMu         sync.RWMutex

	// Context for cleanup
	ctx    context.Context
	cancel context.CancelFunc
}

// WebSocket message structures for Bybit
type BybitWSMessage struct {
	Topic string          `json:"topic"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
	Ts    int64           `json:"ts"`
}

type BybitKlineWS struct {
	Start  int64  `json:"start"`
	End    int64  `json:"end"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
	Symbol string `json:"symbol"`
}

type BybitOrderBookWS struct {
	Symbol string     `json:"s"`
	Bids   [][]string `json:"b"`
	Asks   [][]string `json:"a"`
	Ts     int64      `json:"ts"`
}

// NewBybit creates a new Bybit exchange instance
func NewBybit(apiKey, secret string, testnet bool, logger zerolog.Logger) *BybitExchange {
	var client *bybit.Client
	if testnet {
		client = bybit.NewClient().WithAuth(apiKey, secret).WithBaseURL("https://api-testnet.bybit.com")
	} else {
		client = bybit.NewClient().WithAuth(apiKey, secret)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BybitExchange{
		client:        client,
		logger:        logger.With().Str("exchange", "bybit").Logger(),
		name:          "bybit",
		testnet:       testnet,
		subscriptions: make(map[string]DataHandler),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// GetName returns the exchange name
func (b *BybitExchange) GetName() string {
	return b.name
}

// Connect establishes connection to the exchange
func (b *BybitExchange) Connect(ctx context.Context) error {
	b.logger.Info().Bool("testnet", b.testnet).Msg("Connecting to Bybit")

	// Test REST API connection
	if b.testnet {
		b.logger.Info().Msg("Using Bybit testnet")
	}

	// Initialize WebSocket connection
	if err := b.connectWebSocket(); err != nil {
		b.logger.Error().Err(err).Msg("Failed to connect to Bybit WebSocket")
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	b.connected = true
	b.logger.Info().Msg("Successfully connected to Bybit")
	return nil
}

// connectWebSocket establishes WebSocket connection
func (b *BybitExchange) connectWebSocket() error {
	var wsURL string
	if b.testnet {
		wsURL = "wss://stream-testnet.bybit.com/v5/public/spot"
	} else {
		wsURL = "wss://stream.bybit.com/v5/public/spot"
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	b.wsConnMu.Lock()
	b.wsConn = conn
	b.wsConnMu.Unlock()

	// Start WebSocket message handler
	go b.handleWebSocketMessages()

	b.logger.Info().Str("url", wsURL).Msg("WebSocket connected")
	return nil
}

// handleWebSocketMessages processes incoming WebSocket messages
func (b *BybitExchange) handleWebSocketMessages() {
	defer func() {
		b.wsConnMu.Lock()
		if b.wsConn != nil {
			b.wsConn.Close()
			b.wsConn = nil
		}
		b.wsConnMu.Unlock()
	}()

	for {
		select {
		case <-b.ctx.Done():
			return
		default:
			b.wsConnMu.RLock()
			conn := b.wsConn
			b.wsConnMu.RUnlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				b.logger.Error().Err(err).Msg("WebSocket read error")
				return
			}

			if err := b.processWebSocketMessage(message); err != nil {
				b.logger.Error().Err(err).Msg("Error processing WebSocket message")
			}
		}
	}
}

// processWebSocketMessage parses and routes WebSocket messages
func (b *BybitExchange) processWebSocketMessage(message []byte) error {
	var wsMsg BybitWSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		return fmt.Errorf("failed to unmarshal WebSocket message: %w", err)
	}

	b.subMu.RLock()
	handler, exists := b.subscriptions[wsMsg.Topic]
	b.subMu.RUnlock()

	if !exists {
		return nil // No handler for this topic
	}

	// Parse based on topic type
	if strings.Contains(wsMsg.Topic, "kline") {
		return b.handleKlineMessage(wsMsg, handler)
	} else if strings.Contains(wsMsg.Topic, "orderbook") {
		return b.handleOrderBookMessage(wsMsg, handler)
	}

	return nil
}

// handleKlineMessage processes kline WebSocket messages
func (b *BybitExchange) handleKlineMessage(wsMsg BybitWSMessage, handler DataHandler) error {
	var klineData []BybitKlineWS
	if err := json.Unmarshal(wsMsg.Data, &klineData); err != nil {
		return fmt.Errorf("failed to unmarshal kline data: %w", err)
	}

	for _, k := range klineData {
		open, _ := strconv.ParseFloat(k.Open, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		close, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)

		kline := &Kline{
			Symbol:    k.Symbol,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			Timestamp: time.Unix(k.Start/1000, 0),
			Interval:  b.extractIntervalFromTopic(wsMsg.Topic),
		}

		handler.OnKline(kline)
	}

	return nil
}

// handleOrderBookMessage processes order book WebSocket messages
func (b *BybitExchange) handleOrderBookMessage(wsMsg BybitWSMessage, handler DataHandler) error {
	var obData BybitOrderBookWS
	if err := json.Unmarshal(wsMsg.Data, &obData); err != nil {
		return fmt.Errorf("failed to unmarshal orderbook data: %w", err)
	}

	var bids, asks []OrderBookEntry

	for _, bid := range obData.Bids {
		if len(bid) >= 2 {
			price, _ := strconv.ParseFloat(bid[0], 64)
			quantity, _ := strconv.ParseFloat(bid[1], 64)
			bids = append(bids, OrderBookEntry{Price: price, Quantity: quantity})
		}
	}

	for _, ask := range obData.Asks {
		if len(ask) >= 2 {
			price, _ := strconv.ParseFloat(ask[0], 64)
			quantity, _ := strconv.ParseFloat(ask[1], 64)
			asks = append(asks, OrderBookEntry{Price: price, Quantity: quantity})
		}
	}

	orderBook := &OrderBook{
		Symbol:    obData.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Unix(obData.Ts/1000, 0),
	}

	handler.OnOrderBook(orderBook)
	return nil
}

// extractIntervalFromTopic extracts interval from WebSocket topic
func (b *BybitExchange) extractIntervalFromTopic(topic string) string {
	// Extract interval from topic like "kline.1.BTCUSDT"
	parts := strings.Split(topic, ".")
	if len(parts) >= 2 {
		return b.reverseMapInterval(parts[1])
	}
	return "1m"
}

// Disconnect closes connection to the exchange
func (b *BybitExchange) Disconnect() error {
	b.logger.Info().Msg("Disconnecting from Bybit")

	b.cancel() // Cancel context to stop goroutines

	b.wsConnMu.Lock()
	if b.wsConn != nil {
		b.wsConn.Close()
		b.wsConn = nil
	}
	b.wsConnMu.Unlock()

	b.connected = false
	return nil
}

// IsConnected checks if connected to the exchange
func (b *BybitExchange) IsConnected() bool {
	return b.connected && b.wsConn != nil
}

// SubscribeKlines subscribes to kline data via WebSocket
func (b *BybitExchange) SubscribeKlines(ctx context.Context, symbols []string, interval string, handler DataHandler) error {
	b.logger.Info().
		Strs("symbols", symbols).
		Str("interval", interval).
		Msg("Subscribing to klines")

	// Map common intervals to Bybit format
	bybitInterval := b.mapInterval(interval)
	if bybitInterval == "" {
		return fmt.Errorf("unsupported interval: %s", interval)
	}

	for _, symbol := range symbols {
		topic := fmt.Sprintf("kline.%s.%s", bybitInterval, symbol)

		b.subMu.Lock()
		b.subscriptions[topic] = handler
		b.subMu.Unlock()

		// Send subscription message
		subMsg := map[string]interface{}{
			"op":   "subscribe",
			"args": []string{topic},
		}

		if err := b.sendWebSocketMessage(subMsg); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
		}

		b.logger.Info().Str("topic", topic).Msg("Subscribed to kline")
	}

	return nil
}

// SubscribeOrderBook subscribes to order book data via WebSocket
func (b *BybitExchange) SubscribeOrderBook(ctx context.Context, symbols []string, handler DataHandler) error {
	b.logger.Info().
		Strs("symbols", symbols).
		Msg("Subscribing to order book")

	for _, symbol := range symbols {
		topic := fmt.Sprintf("orderbook.1.%s", symbol)

		b.subMu.Lock()
		b.subscriptions[topic] = handler
		b.subMu.Unlock()

		// Send subscription message
		subMsg := map[string]interface{}{
			"op":   "subscribe",
			"args": []string{topic},
		}

		if err := b.sendWebSocketMessage(subMsg); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
		}

		b.logger.Info().Str("topic", topic).Msg("Subscribed to orderbook")
	}

	return nil
}

// sendWebSocketMessage sends a message via WebSocket
func (b *BybitExchange) sendWebSocketMessage(msg interface{}) error {
	b.wsConnMu.RLock()
	conn := b.wsConn
	b.wsConnMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	return conn.WriteJSON(msg)
}

// mapInterval maps common interval formats to Bybit format
func (b *BybitExchange) mapInterval(interval string) string {
	intervalMap := map[string]string{
		"1m":  "1",
		"3m":  "3",
		"5m":  "5",
		"15m": "15",
		"30m": "30",
		"1h":  "60",
		"2h":  "120",
		"4h":  "240",
		"6h":  "360",
		"12h": "720",
		"1d":  "D",
		"1w":  "W",
		"1M":  "M",
	}

	return intervalMap[interval]
}

// reverseMapInterval maps Bybit interval back to common format
func (b *BybitExchange) reverseMapInterval(bybitInterval string) string {
	reverseMap := map[string]string{
		"1":   "1m",
		"3":   "3m",
		"5":   "5m",
		"15":  "15m",
		"30":  "30m",
		"60":  "1h",
		"120": "2h",
		"240": "4h",
		"360": "6h",
		"720": "12h",
		"D":   "1d",
		"W":   "1w",
		"M":   "1M",
	}

	if mapped, exists := reverseMap[bybitInterval]; exists {
		return mapped
	}
	return bybitInterval
}

// UnsubscribeKlines unsubscribes from kline data
func (b *BybitExchange) UnsubscribeKlines(symbols []string) error {
	b.logger.Info().Strs("symbols", symbols).Msg("Unsubscribing from klines")

	for _, symbol := range symbols {
		// Remove all kline subscriptions for this symbol
		b.subMu.Lock()
		for topic := range b.subscriptions {
			if strings.Contains(topic, "kline") && strings.Contains(topic, symbol) {
				delete(b.subscriptions, topic)

				// Send unsubscribe message
				unsubMsg := map[string]interface{}{
					"op":   "unsubscribe",
					"args": []string{topic},
				}
				b.sendWebSocketMessage(unsubMsg)
			}
		}
		b.subMu.Unlock()
	}

	return nil
}

// UnsubscribeOrderBook unsubscribes from order book data
func (b *BybitExchange) UnsubscribeOrderBook(symbols []string) error {
	b.logger.Info().Strs("symbols", symbols).Msg("Unsubscribing from order book")

	for _, symbol := range symbols {
		topic := fmt.Sprintf("orderbook.1.%s", symbol)

		b.subMu.Lock()
		delete(b.subscriptions, topic)
		b.subMu.Unlock()

		// Send unsubscribe message
		unsubMsg := map[string]interface{}{
			"op":   "unsubscribe",
			"args": []string{topic},
		}
		b.sendWebSocketMessage(unsubMsg)
	}

	return nil
}

// PlaceOrder places a trading order (simplified for testnet)
func (b *BybitExchange) PlaceOrder(ctx context.Context, order *Order) (*Order, error) {
	b.logger.Info().
		Str("symbol", order.Symbol).
		Str("side", order.Side).
		Str("type", order.Type).
		Float64("quantity", order.Quantity).
		Float64("price", order.Price).
		Msg("Placing order")

	// For now, return a mock order since full integration requires proper API setup
	return &Order{
		ID:       fmt.Sprintf("bybit_%d", time.Now().Unix()),
		Symbol:   order.Symbol,
		Side:     order.Side,
		Type:     order.Type,
		Quantity: order.Quantity,
		Price:    order.Price,
		Status:   "submitted",
		Time:     time.Now(),
	}, nil
}

// CancelOrder cancels an existing order
func (b *BybitExchange) CancelOrder(ctx context.Context, symbol, orderID string) error {
	b.logger.Info().
		Str("symbol", symbol).
		Str("order_id", orderID).
		Msg("Cancelling order")

	// Mock implementation
	return nil
}

// GetOrder retrieves order information
func (b *BybitExchange) GetOrder(ctx context.Context, symbol, orderID string) (*Order, error) {
	// Mock implementation
	return &Order{
		ID:       orderID,
		Symbol:   symbol,
		Side:     "buy",
		Type:     "limit",
		Quantity: 1.0,
		Price:    50000.0,
		Status:   "filled",
		Time:     time.Now(),
	}, nil
}

// GetOpenOrders retrieves all open orders for a symbol
func (b *BybitExchange) GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error) {
	// Mock implementation
	return []*Order{}, nil
}

// GetBalances retrieves account balances
func (b *BybitExchange) GetBalances(ctx context.Context) ([]*Balance, error) {
	// Mock implementation for testnet
	return []*Balance{
		{
			Asset:     "BTC",
			Available: 1.0,
			Locked:    0.0,
			Total:     1.0,
		},
		{
			Asset:     "USDT",
			Available: 50000.0,
			Locked:    0.0,
			Total:     50000.0,
		},
	}, nil
}

// GetPositions retrieves account positions (not applicable for spot trading)
func (b *BybitExchange) GetPositions(ctx context.Context) ([]*Position, error) {
	// Spot trading doesn't have positions in the traditional sense
	return []*Position{}, nil
}

// GetKlines retrieves historical kline data
func (b *BybitExchange) GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]*Kline, error) {
	bybitInterval := b.mapInterval(interval)
	if bybitInterval == "" {
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	// Mock implementation - in production, use actual Bybit API
	var klines []*Kline
	now := time.Now()

	for i := limit; i > 0; i-- {
		baseTime := now.Add(-time.Duration(i) * time.Minute)
		basePrice := 50000.0 + float64(i%100)*10

		klines = append(klines, &Kline{
			Symbol:    symbol,
			Open:      basePrice,
			High:      basePrice + 50,
			Low:       basePrice - 50,
			Close:     basePrice + 25,
			Volume:    1000.0 + float64(i*10),
			Timestamp: baseTime,
			Interval:  interval,
		})
	}

	return klines, nil
}

// GetOrderBook retrieves order book data
func (b *BybitExchange) GetOrderBook(ctx context.Context, symbol string, limit int) (*OrderBook, error) {
	// Mock implementation
	return &OrderBook{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids: []OrderBookEntry{
			{Price: 49950.0, Quantity: 1.5},
			{Price: 49940.0, Quantity: 2.0},
		},
		Asks: []OrderBookEntry{
			{Price: 50050.0, Quantity: 1.2},
			{Price: 50060.0, Quantity: 1.8},
		},
	}, nil
}

// GetTicker retrieves ticker information
func (b *BybitExchange) GetTicker(ctx context.Context, symbol string) (*Ticker, error) {
	// Mock implementation
	return &Ticker{
		Symbol:    symbol,
		Price:     50000.0,
		Volume:    1000.0,
		Change:    500.0,
		ChangeP:   1.0,
		Timestamp: time.Now(),
	}, nil
}

// GetExchangeInfo retrieves exchange information
func (b *BybitExchange) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	// Mock implementation
	return &ExchangeInfo{
		Name: b.name,
		Symbols: []*Symbol{
			{
				Name:              "BTCUSDT",
				BaseAsset:         "BTC",
				QuoteAsset:        "USDT",
				Status:            "trading",
				MinOrderSize:      0.001,
				MaxOrderSize:      1000.0,
				MinPrice:          0.01,
				MaxPrice:          1000000.0,
				PricePrecision:    2,
				QuantityPrecision: 6,
			},
			{
				Name:              "ETHUSDT",
				BaseAsset:         "ETH",
				QuoteAsset:        "USDT",
				Status:            "trading",
				MinOrderSize:      0.01,
				MaxOrderSize:      10000.0,
				MinPrice:          0.01,
				MaxPrice:          100000.0,
				PricePrecision:    2,
				QuantityPrecision: 4,
			},
		},
	}, nil
}
