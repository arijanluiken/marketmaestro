package exchanges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// PlaceOrder places a trading order using Bybit V5 API
func (b *BybitExchange) PlaceOrder(ctx context.Context, order *Order) (*Order, error) {
	b.logger.Info().
		Str("symbol", order.Symbol).
		Str("side", order.Side).
		Str("type", order.Type).
		Float64("quantity", order.Quantity).
		Float64("price", order.Price).
		Msg("Placing order")

	// Convert our order type to Bybit format
	orderType := bybit.OrderTypeLimit
	if order.Type == "market" {
		orderType = bybit.OrderTypeMarket
	}

	side := bybit.SideBuy
	if order.Side == "sell" {
		side = bybit.SideSell
	}

	// Prepare order request parameters
	param := bybit.V5CreateOrderParam{
		Category:  bybit.CategoryV5Spot,
		Symbol:    bybit.SymbolV5(order.Symbol),
		Side:      side,
		OrderType: orderType,
		Qty:       fmt.Sprintf("%.8f", order.Quantity),
	}

	// Add price for limit orders
	if order.Type == "limit" {
		priceStr := fmt.Sprintf("%.8f", order.Price)
		param.Price = &priceStr
	}

	// Use the V5 Order service to place the order
	response, err := b.client.V5().Order().CreateOrder(param)
	if err != nil {
		b.logger.Error().Err(err).Msg("Failed to place order")
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	// Return the created order
	return &Order{
		ID:       response.Result.OrderID,
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

	param := bybit.V5CancelOrderParam{
		Category: bybit.CategoryV5Spot,
		Symbol:   bybit.SymbolV5(symbol),
		OrderID:  &orderID,
	}

	_, err := b.client.V5().Order().CancelOrder(param)
	if err != nil {
		b.logger.Error().Err(err).Msg("Failed to cancel order")
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

// GetOrder retrieves order information
func (b *BybitExchange) GetOrder(ctx context.Context, symbol, orderID string) (*Order, error) {
	// Use GetOpenOrders with orderID filter to get specific order
	param := bybit.V5GetOpenOrdersParam{
		Category: bybit.CategoryV5Spot,
		Symbol:   (*bybit.SymbolV5)(&symbol),
		OrderID:  &orderID,
	}

	resp, err := b.client.V5().Order().GetOpenOrders(param)
	if err != nil {
		// Try history orders if not found in open orders
		historyParam := bybit.V5GetHistoryOrdersParam{
			Category: bybit.CategoryV5Spot,
			Symbol:   (*bybit.SymbolV5)(&symbol),
			OrderID:  &orderID,
		}

		resp, err = b.client.V5().Order().GetHistoryOrders(historyParam)
		if err != nil {
			return nil, fmt.Errorf("failed to get order: %w", err)
		}
	}

	if len(resp.Result.List) == 0 {
		return nil, fmt.Errorf("order not found")
	}

	order := resp.Result.List[0]
	return b.convertV5OrderToOrder(&order), nil
}

// GetOpenOrders retrieves all open orders for a symbol (empty symbol gets all orders)
func (b *BybitExchange) GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error) {
	param := bybit.V5GetOpenOrdersParam{
		Category: bybit.CategoryV5Spot,
	}

	// If symbol is provided, filter by symbol
	if symbol != "" {
		param.Symbol = (*bybit.SymbolV5)(&symbol)
	}

	resp, err := b.client.V5().Order().GetOpenOrders(param)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	orders := make([]*Order, 0, len(resp.Result.List))
	for _, order := range resp.Result.List {
		orders = append(orders, b.convertV5OrderToOrder(&order))
	}

	return orders, nil
}

// convertV5OrderToOrder converts a Bybit V5 order to our Order struct
func (b *BybitExchange) convertV5OrderToOrder(v5Order *bybit.V5GetOrder) *Order {
	quantity, _ := strconv.ParseFloat(v5Order.Qty, 64)
	price, _ := strconv.ParseFloat(v5Order.Price, 64)
	createdTime, _ := strconv.ParseInt(v5Order.CreatedTime, 10, 64)

	side := "buy"
	if v5Order.Side == bybit.SideSell {
		side = "sell"
	}

	orderType := "limit"
	if v5Order.OrderType == bybit.OrderTypeMarket {
		orderType = "market"
	}

	status := strings.ToLower(string(v5Order.OrderStatus))

	return &Order{
		ID:       v5Order.OrderID,
		Symbol:   string(v5Order.Symbol),
		Side:     side,
		Type:     orderType,
		Quantity: quantity,
		Price:    price,
		Status:   status,
		Time:     time.Unix(createdTime/1000, 0),
	}
}

// GetBalances retrieves account balances
func (b *BybitExchange) GetBalances(ctx context.Context) ([]*Balance, error) {
	var allBalances []*Balance

	// Try UNIFIED account type first (recommended for V5 API)
	resp, err := b.client.V5().Account().GetWalletBalance(bybit.AccountTypeV5("UNIFIED"), nil)
	if err != nil {
		b.logger.Error().Err(err).Msg("Failed to get UNIFIED wallet balance")
	} else {
		b.logger.Info().Int("accounts", len(resp.Result.List)).Msg("Retrieved UNIFIED wallet balance")
		for _, account := range resp.Result.List {
			b.logger.Info().
				Str("account_type", string(account.AccountType)).
				Int("coins", len(account.Coin)).
				Msg("Processing account")

			for _, coin := range account.Coin {
				// Parse balance fields - handle cases where Free might be empty
				available, _ := strconv.ParseFloat(coin.Free, 64)
				locked, _ := strconv.ParseFloat(coin.Locked, 64)
				total, _ := strconv.ParseFloat(coin.WalletBalance, 64)

				// If Free is empty but WalletBalance has value, use WalletBalance as available
				// This happens in UNIFIED accounts where funds might not be explicitly "free"
				if available == 0 && total > 0 {
					available = total - locked
				}

				// Only include balances that have actual funds
				if total > 0 {
					b.logger.Info().
						Str("asset", string(coin.Coin)).
						Float64("total", total).
						Float64("available", available).
						Float64("locked", locked).
						Str("raw_free", coin.Free).
						Str("raw_wallet", coin.WalletBalance).
						Msg("Found balance")

					allBalances = append(allBalances, &Balance{
						Asset:     string(coin.Coin),
						Available: available,
						Locked:    locked,
						Total:     total,
					})
				}
			}
		}
	}

	b.logger.Info().Int("total_balances", len(allBalances)).Msg("GetBalances completed")
	return allBalances, nil
} // GetPositions retrieves account positions (not applicable for spot trading)
func (b *BybitExchange) GetPositions(ctx context.Context) ([]*Position, error) {
	// Spot trading doesn't have positions in the traditional sense
	return []*Position{}, nil
}

// getUSDIndexPrice fetches the USD index price for a symbol from Bybit testnet
func (b *BybitExchange) getUSDIndexPrice(symbol string) (float64, error) {
	if !b.testnet {
		return 0, nil
	}

	url := fmt.Sprintf("https://api-testnet.bybit.com/v5/market/tickers?category=spot&symbol=%s", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch ticker: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var tickerResponse struct {
		Result struct {
			List []struct {
				UsdIndexPrice string `json:"usdIndexPrice"`
				LastPrice     string `json:"lastPrice"`
			} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &tickerResponse); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(tickerResponse.Result.List) == 0 {
		return 0, fmt.Errorf("no ticker data found")
	}

	usdIndexPrice, err := strconv.ParseFloat(tickerResponse.Result.List[0].UsdIndexPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse USD index price: %w", err)
	}

	return usdIndexPrice, nil
}

// GetKlines retrieves historical kline data
func (b *BybitExchange) GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]*Kline, error) {
	bybitInterval := b.mapIntervalToV5(interval)
	if bybitInterval == "" {
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	// Get spot market data for OHLC structure
	param := bybit.V5GetKlineParam{
		Category: bybit.CategoryV5Spot,
		Symbol:   bybit.SymbolV5(symbol),
		Interval: bybit.Interval(bybitInterval),
		Limit:    &limit,
	}

	resp, err := b.client.V5().Market().GetKline(param)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// Get USD index price for testnet accuracy
	var usdIndexPrice float64
	var spotPrice float64
	if b.testnet && symbol == "BTCUSDT" {
		usdIndexPrice, err = b.getUSDIndexPrice(symbol)
		if err != nil {
			b.logger.Warn().Err(err).Msg("Failed to get USD index price, using spot prices")
		} else if len(resp.Result.List) > 0 {
			spotPrice, _ = strconv.ParseFloat(resp.Result.List[0].Close, 64)

			b.logger.Info().
				Str("symbol", symbol).
				Float64("spot_price", spotPrice).
				Float64("usd_index_price", usdIndexPrice).
				Msg("Retrieved USD index price for testnet accuracy")
		}
	}

	b.logger.Info().
		Str("symbol", symbol).
		Str("category", "spot").
		Int("result_count", len(resp.Result.List)).
		Float64("usd_index_price", usdIndexPrice).
		Msg("Received spot klines from Bybit API")

	var klines []*Kline
	for _, item := range resp.Result.List {
		open, _ := strconv.ParseFloat(item.Open, 64)
		high, _ := strconv.ParseFloat(item.High, 64)
		low, _ := strconv.ParseFloat(item.Low, 64)
		closePrice, _ := strconv.ParseFloat(item.Close, 64)
		volume, _ := strconv.ParseFloat(item.Volume, 64)
		startTime, _ := strconv.ParseInt(item.StartTime, 10, 64)

		// For testnet, adjust prices using USD index price ratio to maintain OHLC structure
		// while providing accurate pricing relative to real market values
		if b.testnet && usdIndexPrice > 0 && spotPrice > 0 {
			ratio := usdIndexPrice / spotPrice
			originalClose := closePrice
			open *= ratio
			high *= ratio
			low *= ratio
			closePrice *= ratio

			b.logger.Debug().
				Str("symbol", symbol).
				Float64("ratio", ratio).
				Float64("original_close", originalClose).
				Float64("adjusted_close", closePrice).
				Msg("Applied USD index price ratio to kline data")
		}

		klines = append(klines, &Kline{
			Symbol:    symbol,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			Timestamp: time.Unix(startTime/1000, 0),
			Interval:  interval,
		})
	}

	return klines, nil
}

// mapIntervalToV5 maps common interval formats to Bybit V5 format
func (b *BybitExchange) mapIntervalToV5(interval string) string {
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
