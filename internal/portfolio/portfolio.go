package portfolio

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for portfolio actor communication
type (
	// Position update messages
	UpdatePositionMsg struct {
		Exchange string
		Symbol   string
		Quantity float64
		Price    float64
		Side     string // "buy" or "sell"
	}

	// Balance update messages
	UpdateBalanceMsg struct {
		Exchange string
		Asset    string
		Amount   float64
	}

	// Trade execution messages
	TradeExecutedMsg struct {
		Trade Trade
	}

	// Exchange synchronization messages
	SyncWithExchangeMsg   struct{}
	RequestBalancesMsg    struct{}
	RequestPositionsMsg   struct{}
	UpdateMarketPricesMsg struct {
		Prices map[string]float64 // symbol -> current price
	}
	SetExchangeActorMsg struct {
		ExchangeActorPID *actor.PID
	}

	// Portfolio queries
	GetPositionsMsg   struct{}
	GetBalancesMsg    struct{}
	GetPerformanceMsg struct{}
	StatusMsg         struct{}

	// Portfolio responses
	PositionsResponse struct {
		Positions []Position `json:"positions"`
	}

	BalancesResponse struct {
		Balances []Balance `json:"balances"`
	}

	PerformanceResponse struct {
		TotalValue    float64 `json:"total_value"`
		AvailableCash float64 `json:"available_cash"`
		UnrealizedPnL float64 `json:"unrealized_pnl"`
		RealizedPnL   float64 `json:"realized_pnl"`
		DailyPnL      float64 `json:"daily_pnl"`
		WeeklyPnL     float64 `json:"weekly_pnl"`
		MonthlyPnL    float64 `json:"monthly_pnl"`
	}
)

// Portfolio data structures
type Position struct {
	Exchange      string    `json:"exchange"`
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AveragePrice  float64   `json:"average_price"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Balance struct {
	Exchange  string    `json:"exchange"`
	Asset     string    `json:"asset"`
	Available float64   `json:"available"`
	Locked    float64   `json:"locked"`
	Total     float64   `json:"total"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Trade struct {
	ID        string    `json:"id"`
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Fee       float64   `json:"fee"`
	Timestamp time.Time `json:"timestamp"`
}

// PortfolioActor manages positions, balances, and performance tracking
type PortfolioActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
	positions    map[string]*Position // key: exchange:symbol
	balances     map[string]*Balance  // key: exchange:asset
	trades       []Trade
	pnlHistory   map[string]float64 // key: date (YYYY-MM-DD)

	// Exchange actor reference for real-time data
	exchangeActorPID *actor.PID

	// Market data for portfolio valuation
	currentPrices map[string]float64 // symbol -> current price

	// Performance tracking
	lastUpdateTime time.Time
	syncInterval   time.Duration
}

// New creates a new portfolio actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *PortfolioActor {
	return &PortfolioActor{
		exchangeName:  exchangeName,
		config:        cfg,
		db:            db,
		logger:        logger,
		positions:     make(map[string]*Position),
		balances:      make(map[string]*Balance),
		trades:        make([]Trade, 0),
		pnlHistory:    make(map[string]float64),
		currentPrices: make(map[string]float64),
		syncInterval:  time.Minute * 5, // Sync with exchange every 5 minutes
	}
}

// SetExchangeActor sets the reference to the exchange actor
func (p *PortfolioActor) SetExchangeActor(exchangeActorPID *actor.PID) {
	p.exchangeActorPID = exchangeActorPID
	p.logger.Info().Msg("Exchange actor reference set")
}

// Receive handles incoming messages
func (p *PortfolioActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		p.onStarted(ctx)
	case actor.Stopped:
		p.onStopped(ctx)
	case UpdatePositionMsg:
		p.onUpdatePosition(ctx, msg)
	case UpdateBalanceMsg:
		p.onUpdateBalance(ctx, msg)
	case TradeExecutedMsg:
		p.onTradeExecuted(ctx, msg)
	case SyncWithExchangeMsg:
		p.onSyncWithExchange(ctx)
	case RequestBalancesMsg:
		p.onRequestBalances(ctx)
	case RequestPositionsMsg:
		p.onRequestPositions(ctx)
	case UpdateMarketPricesMsg:
		p.onUpdateMarketPrices(ctx, msg)
	case SetExchangeActorMsg:
		p.onSetExchangeActor(ctx, msg)
	case GetPositionsMsg:
		p.onGetPositions(ctx)
	case GetBalancesMsg:
		p.onGetBalances(ctx)
	case GetPerformanceMsg:
		p.onGetPerformance(ctx)
	case StatusMsg:
		p.onStatus(ctx)
	default:
		p.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (p *PortfolioActor) onStarted(ctx *actor.Context) {
	p.logger.Info().
		Str("exchange", p.exchangeName).
		Msg("Portfolio actor started")

	// Initialize with real data from exchange
	p.onSyncWithExchange(ctx)

	// Start periodic portfolio updates
	p.schedulePeriodicUpdates(ctx)

	// Start periodic exchange synchronization
	p.scheduleExchangeSync(ctx)
}

func (p *PortfolioActor) onStopped(ctx *actor.Context) {
	p.logger.Info().
		Str("exchange", p.exchangeName).
		Msg("Portfolio actor stopped")
}

func (p *PortfolioActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":       p.exchangeName,
		"timestamp":      time.Now(),
		"positions":      len(p.positions),
		"balances":       len(p.balances),
		"trades":         len(p.trades),
		"total_value":    p.calculateTotalValue(),
		"unrealized_pnl": p.calculateUnrealizedPnL(),
	}

	ctx.Respond(status)
}

func (p *PortfolioActor) onUpdatePosition(ctx *actor.Context, msg UpdatePositionMsg) {
	key := fmt.Sprintf("%s:%s", msg.Exchange, msg.Symbol)

	position, exists := p.positions[key]
	if !exists {
		// Create new position
		position = &Position{
			Exchange:     msg.Exchange,
			Symbol:       msg.Symbol,
			Quantity:     0,
			AveragePrice: 0,
			UpdatedAt:    time.Now(),
		}
		p.positions[key] = position
	}

	// Update position based on trade
	if msg.Side == "buy" {
		// Calculate new average price
		totalValue := position.Quantity * position.AveragePrice
		newTotalValue := totalValue + (msg.Quantity * msg.Price)
		newQuantity := position.Quantity + msg.Quantity

		if newQuantity > 0 {
			position.AveragePrice = newTotalValue / newQuantity
		}
		position.Quantity = newQuantity
	} else if msg.Side == "sell" {
		position.Quantity -= msg.Quantity

		// If position is closed, reset average price
		if position.Quantity <= 0 {
			position.Quantity = 0
			position.AveragePrice = 0
		}
	}

	position.CurrentPrice = msg.Price
	position.UpdatedAt = time.Now()

	// Calculate unrealized PnL
	if position.Quantity > 0 {
		position.UnrealizedPnL = (position.CurrentPrice - position.AveragePrice) * position.Quantity
	} else {
		position.UnrealizedPnL = 0
	}

	// Record trade
	trade := Trade{
		ID:        fmt.Sprintf("%s_%d", key, time.Now().Unix()),
		Exchange:  msg.Exchange,
		Symbol:    msg.Symbol,
		Side:      msg.Side,
		Quantity:  msg.Quantity,
		Price:     msg.Price,
		Fee:       msg.Quantity * msg.Price * 0.001, // Assume 0.1% fee
		Timestamp: time.Now(),
	}
	p.trades = append(p.trades, trade)

	p.logger.Info().
		Str("exchange", msg.Exchange).
		Str("symbol", msg.Symbol).
		Str("side", msg.Side).
		Float64("quantity", msg.Quantity).
		Float64("price", msg.Price).
		Float64("new_position", position.Quantity).
		Float64("avg_price", position.AveragePrice).
		Float64("unrealized_pnl", position.UnrealizedPnL).
		Msg("Position updated")
}

func (p *PortfolioActor) onUpdateBalance(ctx *actor.Context, msg UpdateBalanceMsg) {
	key := fmt.Sprintf("%s:%s", msg.Exchange, msg.Asset)

	balance, exists := p.balances[key]
	if !exists {
		balance = &Balance{
			Exchange:  msg.Exchange,
			Asset:     msg.Asset,
			Available: 0,
			Locked:    0,
			UpdatedAt: time.Now(),
		}
		p.balances[key] = balance
	}

	balance.Available = msg.Amount
	balance.Total = balance.Available + balance.Locked
	balance.UpdatedAt = time.Now()

	p.logger.Debug().
		Str("exchange", msg.Exchange).
		Str("asset", msg.Asset).
		Float64("amount", msg.Amount).
		Msg("Balance updated")
}

func (p *PortfolioActor) onGetPositions(ctx *actor.Context) {
	positions := make([]Position, 0, len(p.positions))
	for _, position := range p.positions {
		if position.Quantity > 0 { // Only return non-zero positions
			positions = append(positions, *position)
		}
	}

	response := PositionsResponse{
		Positions: positions,
	}

	ctx.Respond(response)
}

func (p *PortfolioActor) onGetBalances(ctx *actor.Context) {
	balances := make([]Balance, 0, len(p.balances))
	for _, balance := range p.balances {
		if balance.Total > 0 { // Only return non-zero balances
			balances = append(balances, *balance)
		}
	}

	response := BalancesResponse{
		Balances: balances,
	}

	ctx.Respond(response)
}

func (p *PortfolioActor) onGetPerformance(ctx *actor.Context) {
	totalValue := p.calculateTotalValue()
	availableCash := p.calculateAvailableCash()
	unrealizedPnL := p.calculateUnrealizedPnL()
	realizedPnL := p.calculateRealizedPnL()

	response := PerformanceResponse{
		TotalValue:    totalValue,
		AvailableCash: availableCash,
		UnrealizedPnL: unrealizedPnL,
		RealizedPnL:   realizedPnL,
		DailyPnL:      p.calculateDailyPnL(),
		WeeklyPnL:     p.calculateWeeklyPnL(),
		MonthlyPnL:    p.calculateMonthlyPnL(),
	}

	ctx.Respond(response)
}

func (p *PortfolioActor) calculateTotalValue() float64 {
	total := 0.0

	// Add cash balances
	for _, balance := range p.balances {
		if balance.Asset == "USDT" || balance.Asset == "USD" {
			total += balance.Total
		}
	}

	// Add position values
	for _, position := range p.positions {
		if position.Quantity > 0 {
			total += position.Quantity * position.CurrentPrice
		}
	}

	return total
}

func (p *PortfolioActor) calculateAvailableCash() float64 {
	cash := 0.0

	for _, balance := range p.balances {
		if balance.Asset == "USDT" || balance.Asset == "USD" {
			cash += balance.Available
		}
	}

	return cash
}

func (p *PortfolioActor) calculateUnrealizedPnL() float64 {
	pnl := 0.0

	for _, position := range p.positions {
		pnl += position.UnrealizedPnL
	}

	return pnl
}

func (p *PortfolioActor) calculateRealizedPnL() float64 {
	// Calculate from trades (simplified)
	pnl := 0.0

	for _, trade := range p.trades {
		if trade.Side == "sell" {
			// Simplified: assume all sells are profitable
			pnl += trade.Quantity * trade.Price * 0.01 // 1% profit assumption
		}
	}

	return pnl
}

func (p *PortfolioActor) calculateDailyPnL() float64 {
	today := time.Now().Format("2006-01-02")
	return p.pnlHistory[today]
}

func (p *PortfolioActor) calculateWeeklyPnL() float64 {
	weekAgo := time.Now().AddDate(0, 0, -7)
	total := 0.0

	for dateStr, pnl := range p.pnlHistory {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if date.After(weekAgo) {
			total += pnl
		}
	}

	return total
}

func (p *PortfolioActor) calculateMonthlyPnL() float64 {
	monthAgo := time.Now().AddDate(0, -1, 0)
	total := 0.0

	for dateStr, pnl := range p.pnlHistory {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if date.After(monthAgo) {
			total += pnl
		}
	}

	return total
}

func (p *PortfolioActor) schedulePeriodicUpdates(ctx *actor.Context) {
	// Schedule daily PnL calculation
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			p.updateDailyPnL()
		}
	}()
}

func (p *PortfolioActor) scheduleExchangeSync(ctx *actor.Context) {
	// Schedule periodic synchronization with exchange
	go func() {
		ticker := time.NewTicker(p.syncInterval)
		defer ticker.Stop()

		for range ticker.C {
			ctx.Send(ctx.PID(), SyncWithExchangeMsg{})
		}
	}()
}

// Exchange interaction methods

func (p *PortfolioActor) onTradeExecuted(ctx *actor.Context, msg TradeExecutedMsg) {
	// Add trade to history
	p.trades = append(p.trades, msg.Trade)

	// Update position based on trade
	key := fmt.Sprintf("%s:%s", msg.Trade.Exchange, msg.Trade.Symbol)
	position, exists := p.positions[key]

	if !exists {
		// Create new position
		position = &Position{
			Exchange:      msg.Trade.Exchange,
			Symbol:        msg.Trade.Symbol,
			Quantity:      0,
			AveragePrice:  0,
			CurrentPrice:  msg.Trade.Price,
			UnrealizedPnL: 0,
			UpdatedAt:     time.Now(),
		}
		p.positions[key] = position
	}

	// Update position quantity and average price
	if msg.Trade.Side == "buy" {
		totalValue := position.Quantity*position.AveragePrice + msg.Trade.Quantity*msg.Trade.Price
		position.Quantity += msg.Trade.Quantity
		if position.Quantity > 0 {
			position.AveragePrice = totalValue / position.Quantity
		}
	} else { // sell
		position.Quantity -= msg.Trade.Quantity
		if position.Quantity < 0 {
			position.Quantity = 0 // Prevent negative positions in spot trading
		}
	}

	position.UpdatedAt = time.Now()

	// Update balance (subtract fees and trade amount)
	quoteAsset := "USDT" // Assuming USDT as quote asset for simplicity
	balanceKey := fmt.Sprintf("%s:%s", msg.Trade.Exchange, quoteAsset)
	balance, exists := p.balances[balanceKey]
	if exists {
		if msg.Trade.Side == "buy" {
			balance.Available -= msg.Trade.Quantity * msg.Trade.Price
		} else {
			balance.Available += msg.Trade.Quantity * msg.Trade.Price
		}
		balance.Available -= msg.Trade.Fee
		balance.Total = balance.Available + balance.Locked
		balance.UpdatedAt = time.Now()
	}

	p.logger.Info().
		Str("exchange", msg.Trade.Exchange).
		Str("symbol", msg.Trade.Symbol).
		Str("side", msg.Trade.Side).
		Float64("quantity", msg.Trade.Quantity).
		Float64("price", msg.Trade.Price).
		Msg("Trade executed and portfolio updated")
}

func (p *PortfolioActor) onSyncWithExchange(ctx *actor.Context) {
	if p.exchangeActorPID == nil {
		p.logger.Warn().Msg("Exchange actor not set, cannot sync")
		return
	}

	p.logger.Info().Msg("Synchronizing portfolio with exchange")

	// Request current balances from exchange
	ctx.Send(p.exchangeActorPID, GetBalancesMsg{})

	// Request current positions from exchange
	ctx.Send(p.exchangeActorPID, GetPositionsMsg{})

	p.lastUpdateTime = time.Now()
}

func (p *PortfolioActor) onRequestBalances(ctx *actor.Context) {
	if p.exchangeActorPID == nil {
		p.logger.Warn().Msg("Exchange actor not set, cannot request balances")
		return
	}

	p.logger.Debug().Msg("Requesting balances from exchange")
	ctx.Send(p.exchangeActorPID, GetBalancesMsg{})
}

func (p *PortfolioActor) onRequestPositions(ctx *actor.Context) {
	if p.exchangeActorPID == nil {
		p.logger.Warn().Msg("Exchange actor not set, cannot request positions")
		return
	}

	p.logger.Debug().Msg("Requesting positions from exchange")
	ctx.Send(p.exchangeActorPID, GetPositionsMsg{})
}

func (p *PortfolioActor) onUpdateMarketPrices(ctx *actor.Context, msg UpdateMarketPricesMsg) {
	p.logger.Debug().
		Int("price_count", len(msg.Prices)).
		Msg("Updating market prices")

	// Update current prices
	for symbol, price := range msg.Prices {
		p.currentPrices[symbol] = price

		// Update position current prices and unrealized PnL
		key := fmt.Sprintf("%s:%s", p.exchangeName, symbol)
		if position, exists := p.positions[key]; exists {
			position.CurrentPrice = price
			position.UnrealizedPnL = (price - position.AveragePrice) * position.Quantity
			position.UpdatedAt = time.Now()

			p.logger.Debug().
				Str("symbol", symbol).
				Float64("current_price", price).
				Float64("unrealized_pnl", position.UnrealizedPnL).
				Msg("Position PnL updated")
		}
	}

	// Update daily PnL
	p.updateDailyPnL()
}

func (p *PortfolioActor) onSetExchangeActor(ctx *actor.Context, msg SetExchangeActorMsg) {
	p.exchangeActorPID = msg.ExchangeActorPID
	p.logger.Info().Msg("Exchange actor reference set via message")

	// Immediately sync with exchange
	ctx.Send(ctx.PID(), SyncWithExchangeMsg{})
}

func (p *PortfolioActor) updateDailyPnL() {
	today := time.Now().Format("2006-01-02")
	unrealizedPnL := p.calculateUnrealizedPnL()

	p.pnlHistory[today] = unrealizedPnL

	p.logger.Info().
		Str("date", today).
		Float64("pnl", unrealizedPnL).
		Msg("Daily PnL updated")
}
