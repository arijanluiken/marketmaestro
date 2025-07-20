# MarketMaestro Internal Architecture

## Table of Contents
- [System Overview](#system-overview)
- [Actor System Design](#actor-system-design)
- [Message Passing Architecture](#message-passing-architecture)
- [Exchange Integration Layer](#exchange-integration-layer)
- [Strategy Engine Architecture](#strategy-engine-architecture)
- [Data Management](#data-management)
- [API & Web Interface](#api--web-interface)
- [Security & Risk Management](#security--risk-management)
- [Deployment Architecture](#deployment-architecture)
- [Development Guidelines](#development-guidelines)

## System Overview

MarketMaestro is a sophisticated cryptocurrency trading bot built on the **Hollywood Actor Framework**, implementing a pure message-passing architecture. The system is designed for high reliability, scalability, and fault tolerance through strict actor isolation and event-driven processing.

### Core Architectural Principles

1. **Actor-Based Concurrency**: All business logic runs within isolated actors that communicate via immutable messages
2. **Message Passing**: No shared state between components; all communication happens through structured messages
3. **Event-Driven Processing**: Reactive system that responds to market events and user actions
4. **Fault Isolation**: Actor supervision ensures that failures in one component don't cascade to others
5. **Modular Design**: Clear separation of concerns with well-defined interfaces between components

### Technology Stack

- **Runtime**: Go 1.24.4+
- **Actor Framework**: [Hollywood](https://github.com/anthdm/hollywood) by anthdm
- **Web Framework**: Chi router with OpenAPI 3.0
- **Database**: SQLite with automated migrations
- **Scripting**: Starlark (Python-like) for strategy development
- **Real-time Data**: WebSocket connections to exchanges
- **Logging**: Structured logging with zerolog

## Actor System Design

The system implements a hierarchical actor architecture with clear supervision and communication patterns.

### Actor Hierarchy

```
🎭 Supervisor Actor (Root)
├── 🌐 API Actor (REST Server)
├── 🖥️ UI Actor (Web Interface)
└── 🏦 Exchange Actors (Per Exchange: Bybit, Bitvavo)
    ├── 🧠 Strategy Actors (Per Trading Pair/Strategy)
    ├── 📋 Order Manager Actor (Order Execution)
    ├── 🛡️ Risk Manager Actor (Risk Controls)
    ├── 💼 Portfolio Actor (Balance Tracking)
    ├── ⚙️ Settings Actor (Configuration)
    └── ⚖️ Rebalance Actor (Portfolio Rebalancing)
```

### Actor Responsibilities

#### Supervisor Actor (`internal/supervisor/supervisor.go`)
- **Role**: Root actor managing system lifecycle
- **Responsibilities**:
  - Initialize and coordinate all child actors
  - Load configuration and establish database connections
  - Handle graceful shutdown and error recovery
  - Manage inter-actor communication setup
- **Key Messages**: `StartMessage`, `StopMessage`, `StatusMessage`, `RegisterExchange`

#### Exchange Actor (`internal/exchange/exchange.go`)
- **Role**: Manages all exchange-specific operations
- **Responsibilities**:
  - Establish and maintain exchange WebSocket connections
  - Distribute market data to strategy actors
  - Coordinate order execution through child actors
  - Handle exchange-specific configuration and errors
- **Child Actors**: Strategy, Order Manager, Risk Manager, Portfolio, Settings, Rebalance
- **Key Messages**: `ConnectMessage`, `KlineDataMsg`, `OrderBookDataMsg`, `SubscribeKlinesMsg`

#### Strategy Actor (`internal/strategy/strategy.go`)
- **Role**: Executes Starlark-based trading strategies
- **Responsibilities**:
  - Load and execute strategy scripts from `/strategy/*.star` files
  - Process market data (klines, orderbook, ticker)
  - Generate trading signals based on strategy logic
  - Maintain strategy-specific state and buffers
- **Starlark Integration**: 25+ technical indicators, safe execution environment
- **Key Messages**: `KlineDataMsg`, `OrderBookDataMsg`, `ExecuteStrategyMsg`

#### Order Manager Actor (`internal/order/order.go`)
- **Role**: Handles all order placement and execution
- **Responsibilities**:
  - Process order requests from strategies
  - Manage order lifecycle (pending, filled, cancelled)
  - Support advanced order types (stop-loss, trailing stops)
  - Coordinate with exchange APIs for order execution
- **Order Types**: Market, Limit, Stop Market, Stop Limit, Trailing Stop
- **Key Messages**: `PlaceOrderMsg`, `CancelOrderMsg`, `ModifyOrderMsg`

#### Risk Manager Actor (`internal/risk/risk.go`)
- **Role**: Enforces risk controls and position limits
- **Responsibilities**:
  - Validate orders against risk parameters
  - Monitor portfolio exposure and concentration
  - Calculate Value at Risk (VaR) and drawdown metrics
  - Enforce position sizing and leverage limits
- **Risk Metrics**: Max drawdown, VaR95, position concentration, leverage ratio
- **Key Messages**: `ValidateOrderMsg`, `GetRiskMetricsMsg`, `UpdatePortfolioValueMsg`

#### Portfolio Actor (`internal/portfolio/portfolio.go`)
- **Role**: Tracks account balances and positions
- **Responsibilities**:
  - Maintain real-time balance and position data
  - Calculate P&L (realized and unrealized)
  - Sync with exchange account information
  - Provide portfolio performance metrics
- **Tracking**: Balances, positions, trades, performance metrics
- **Key Messages**: `UpdatePositionMsg`, `UpdateBalanceMsg`, `GetPerformanceMsg`

#### Settings Actor (`internal/settings/settings.go`)
- **Role**: Manages persistent configuration
- **Responsibilities**:
  - Store and retrieve actor configuration
  - Handle runtime configuration updates
  - Persist settings to database
  - Provide configuration to other actors
- **Storage**: Key-value pairs in SQLite database
- **Key Messages**: `SetSettingMsg`, `GetSettingMsg`, `LoadSettingsMsg`

## Message Passing Architecture

### Message Design Patterns

All messages follow consistent naming and structure conventions:

```go
// Message naming convention: {Action}{Entity}Msg
type KlineDataMsg struct {
    Kline *exchanges.Kline
}

type PlaceOrderMsg struct {
    Symbol   string
    Side     string
    Type     string
    Quantity float64
    Price    float64
    Reason   string
}
```

### Communication Patterns

#### 1. **Market Data Flow**
```
Exchange WebSocket → Exchange Actor → Strategy Actors
                                  ↓
                            Signal Generation
                                  ↓
                              Risk Manager → Order Manager → Exchange API
```

#### 2. **Order Execution Flow**
```
Strategy Actor → Order Manager → Risk Manager → Exchange Actor → Exchange API
                                      ↓
                               Portfolio Actor (update positions)
```

#### 3. **Status and Monitoring Flow**
```
API Actor → Exchange Actor → Child Actors (gather status)
               ↓
         Aggregate Response → API Response
```

### Message Categories

- **Data Messages**: Market data distribution (`KlineDataMsg`, `OrderBookDataMsg`)
- **Command Messages**: Action requests (`PlaceOrderMsg`, `CancelOrderMsg`)
- **Query Messages**: Information requests (`StatusMsg`, `GetBalancesMsg`)
- **Notification Messages**: Event notifications (`OrderUpdateMsg`, `BalanceUpdateMsg`)
- **Configuration Messages**: Settings management (`SetSettingMsg`, `LoadSettingsMsg`)

### Order Book Data Handling

The system implements robust order book processing that gracefully handles real-world market conditions:

#### Partial Order Book Support
- **Low Liquidity Handling**: Accepts order books with only bids or only asks (common in crypto markets)
- **Price Calculation**: Uses bid price when asks are unavailable, ask price when bids are unavailable
- **Fallback Logic**: Mid-price calculation when both sides are available

#### Exchange Actor Processing
```go
func (e *ExchangeActor) OnOrderBook(orderBook *exchanges.OrderBook) {
    // Only warn for completely empty order books
    if len(orderBook.Bids) == 0 && len(orderBook.Asks) == 0 {
        e.logger.Warn().Msg("Completely empty order book received")
        return
    }
    
    // Debug log for partial order books (normal during low liquidity)
    if len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
        e.logger.Debug().Msg("Partial order book received (low liquidity)")
    }
    
    // Calculate appropriate price for order management
    var priceForUpdate float64
    if len(orderBook.Bids) > 0 && len(orderBook.Asks) > 0 {
        priceForUpdate = (orderBook.Bids[0].Price + orderBook.Asks[0].Price) / 2
    } else if len(orderBook.Bids) > 0 {
        priceForUpdate = orderBook.Bids[0].Price
    } else if len(orderBook.Asks) > 0 {
        priceForUpdate = orderBook.Asks[0].Price
    }
}
```

#### Strategy Actor Processing
- **Flexible Validation**: Strategies receive partial order books and decide how to handle them
- **Context Awareness**: Order book state is available in strategy context for decision making
- **Risk Management**: Strategies can implement custom logic for low-liquidity scenarios

## Exchange Integration Layer

### Exchange Interface (`pkg/exchanges/interface.go`)

The system uses a unified interface pattern to abstract exchange-specific implementations:

```go
type Exchange interface {
    // Connectivity
    Connect(ctx context.Context) error
    Disconnect() error
    IsConnected() bool

    // Data Streaming
    SubscribeKlines(ctx context.Context, symbols []string, interval string, handler DataHandler) error
    SubscribeOrderBook(ctx context.Context, symbols []string, handler DataHandler) error

    // Trading Operations
    PlaceOrder(ctx context.Context, order *Order) (*Order, error)
    CancelOrder(ctx context.Context, symbol, orderID string) error
    GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)

    // Account Information
    GetBalances(ctx context.Context) ([]*Balance, error)
    GetPositions(ctx context.Context) ([]*Position, error)
}
```

### Factory Pattern (`pkg/exchanges/factory.go`)

Exchange instances are created through a factory pattern:

```go
type Factory struct {
    config *config.Config
    logger zerolog.Logger
}

func (f *Factory) CreateExchange(name string) (Exchange, error) {
    switch name {
    case "bybit":
        return NewBybitExchange(f.config, f.logger)
    case "bitvavo":
        return NewBitvavoExchange(f.config, f.logger)
    default:
        return nil, fmt.Errorf("unsupported exchange: %s", name)
    }
}
```

### Current Exchange Implementations

#### Bybit Exchange (`pkg/exchanges/bybit.go`)
- **API**: REST API for trading operations
- **WebSocket**: Real-time market data feeds
- **Features**: Spot and derivatives trading, testnet support
- **Authentication**: API key and secret-based

#### Bitvavo Exchange (`pkg/exchanges/bitvavo.go`)
- **API**: REST API for European market access
- **WebSocket**: Real-time price and orderbook data
- **Features**: Spot trading, EUR pairs
- **Authentication**: API key and secret-based

### Data Structures

```go
type Kline struct {
    Symbol    string
    Timestamp time.Time
    Open      float64
    High      float64
    Low       float64
    Close     float64
    Volume    float64
    Interval  string
}

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
```

## Strategy Engine Architecture

### Starlark Integration

The strategy engine uses Google's Starlark language (Python-like syntax) for safe, sandboxed strategy execution:

#### Required Strategy Functions

```python
def settings():
    """Return strategy configuration"""
    return {
        "interval": "1m",     # Required: Data interval
        "period": 14,         # Custom parameters
        "threshold": 0.02
    }

def on_kline(kline):
    """Process new candlestick data"""
    # Strategy logic here
    if should_buy():
        return {"action": "buy", "quantity": 0.01}
    return {"action": "hold", "quantity": 0.0}

def on_orderbook(orderbook):
    """Process orderbook updates"""
    return {"action": "hold", "quantity": 0.0}

def on_ticker(ticker):
    """Process ticker updates"""
    return {"action": "hold", "quantity": 0.0}
```

#### Optional Lifecycle Functions

```python
def on_start():
    """Called when strategy starts"""
    log("Strategy initialized")

def on_stop():
    """Called when strategy stops"""
    log("Strategy shutting down")
```

### Technical Indicators Library

The engine provides 25+ built-in technical indicators accessible from Starlark:

#### Trend Indicators
- `sma(prices, period)` - Simple Moving Average
- `ema(prices, period)` - Exponential Moving Average
- `wma(prices, period)` - Weighted Moving Average
- `macd(prices, fast, slow, signal)` - MACD Oscillator

#### Momentum Indicators
- `rsi(prices, period)` - Relative Strength Index
- `stoch(highs, lows, closes, k_period)` - Stochastic Oscillator
- `williams_r(highs, lows, closes, period)` - Williams %R

#### Volatility Indicators
- `bollinger_bands(prices, period, std_dev)` - Bollinger Bands
- `atr(highs, lows, closes, period)` - Average True Range
- `keltner_channels(highs, lows, closes, period)` - Keltner Channels

#### Volume Indicators
- `obv(closes, volumes)` - On-Balance Volume
- `mfi(highs, lows, closes, volumes, period)` - Money Flow Index

### Strategy Engine Implementation (`internal/strategy/engine.go`)

```go
type StrategyEngine struct {
    logger     zerolog.Logger
    thread     *starlark.Thread
    globals    starlark.StringDict
    callbacks  *StrategyCallbacks
}

type StrategyCallbacks struct {
    HasOnKline     bool
    HasOnOrderbook bool
    HasOnTicker    bool
    HasSettings    bool
    HasOnStart     bool
    HasOnStop      bool
}
```

### Strategy State Management

- **Persistent State**: Strategies maintain state in the `state` dictionary
- **Buffer Management**: Automatic management of price/volume buffers
- **Configuration**: Runtime access to strategy configuration parameters
- **Logging**: Structured logging available within strategies

## Data Management

### Database Architecture

#### SQLite Schema
The system uses SQLite with automated migrations for data persistence:

```sql
-- Core tables
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    actor_type TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id TEXT PRIMARY KEY,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    type TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL,
    status TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE positions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    quantity REAL NOT NULL,
    average_price REAL NOT NULL,
    current_price REAL,
    unrealized_pnl REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### Migration System (`pkg/database/migrations/`)
- **Automated Migrations**: Run on application startup
- **Version Control**: Sequential migration files with timestamps
- **Rollback Support**: Down migrations for schema rollbacks
- **Migration Tracking**: Built-in migration history table

### Configuration Management (`pkg/config/config.go`)

#### Dual Configuration System

**Environment Variables (`.env`)**:
```env
# Exchange API Keys
BYBIT_API_KEY=your_api_key
BYBIT_SECRET=your_secret_key
BYBIT_TESTNET=true

# Server Configuration
API_PORT=8080
UI_PORT=8081
LOG_LEVEL=info
```

**YAML Configuration (`config.yaml`)**:
```yaml
database:
  path: "./marketmaestro.db"

api:
  port: 8080
  timeout: 30s

exchanges:
  bybit:
    enabled: true
    pairs:
      - symbol: "BTCUSDT"
        strategies:
          - name: "simple_sma"
            config:
              short_period: 10
              long_period: 20
              position_size: 0.01
              interval: "1m"
```

#### Configuration Loading Process

```go
func Load() (*Config, error) {
    // 1. Load YAML configuration
    yamlConfig := loadYAMLConfig()
    
    // 2. Override with environment variables
    envConfig := loadEnvConfig()
    
    // 3. Merge configurations (env takes precedence)
    finalConfig := mergeConfigs(yamlConfig, envConfig)
    
    // 4. Validate configuration
    return validateConfig(finalConfig)
}
```

### State Persistence

#### Actor State Management
- **Settings Actor**: Persists configuration changes to database
- **Portfolio Actor**: Tracks account state and position history
- **Order Manager**: Maintains order history and status
- **Strategy Actors**: Can persist strategy-specific state

#### Database Connection Patterns
```go
// Database injection into actors
func (s *Supervisor) startExchangeActor(ctx *actor.Context, exchangeName string) {
    exchangeActor := exchange.New(
        exchangeName,
        s.config,
        s.db,  // Inject database connection
        s.logger,
    )
    ctx.SpawnChild(func() actor.Receiver { return exchangeActor }, "exchange_"+exchangeName)
}
```

## API & Web Interface

### REST API Architecture (`internal/api/`)

#### Chi Router Configuration
```go
func (a *APIActor) setupRoutes() *chi.Mux {
    r := chi.NewRouter()
    
    // Middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(30 * time.Second))
    
    // API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Get("/health", a.handleHealth)
        r.Get("/openapi.json", a.handleOpenAPI)
        r.Get("/exchanges", a.handleExchanges)
        r.Get("/strategies", a.handleStrategies)
        r.Get("/portfolio", a.handlePortfolio)
        r.Get("/orders", a.handleOrders)
        r.Post("/orders", a.handlePlaceOrder)
    })
    
    return r
}
```

#### API-Actor Communication Pattern
```go
func (a *APIActor) handlePortfolio(w http.ResponseWriter, r *http.Request) {
    // Request data from portfolio actor via message passing
    response, err := a.engine.Request(a.portfolioPID, portfolio.GetPerformanceMsg{}, 5*time.Second)
    if err != nil {
        http.Error(w, "Portfolio request failed", http.StatusInternalServerError)
        return
    }
    
    // Send structured JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(APIResponse{
        Status: "success",
        Data:   response,
        Timestamp: time.Now(),
    })
}
```

#### OpenAPI 3.0 Specification
- **Automated Documentation**: Generated OpenAPI spec available at `/api/v1/openapi.json`
- **Type Safety**: Request/response models defined in Go structs
- **Validation**: Automatic request validation and error handling

### Web Interface (`internal/ui/`)

#### Embedded Asset System
```go
//go:embed assets/*
var assetsFS embed.FS

func (u *UIActor) setupRoutes() *chi.Mux {
    r := chi.NewRouter()
    
    // Serve embedded static assets
    r.Handle("/assets/*", http.StripPrefix("/assets/", 
        http.FileServer(http.FS(assetsFS))))
    
    // Template-based pages
    r.Get("/", u.handleDashboard)
    r.Get("/strategies", u.handleStrategies)
    r.Get("/portfolio", u.handlePortfolio)
    
    return r
}
```

#### Template System
- **Base Template**: Shared layout with navigation and common elements
- **Page Templates**: Specific templates for different sections
- **Component System**: Reusable UI components for consistency
- **Real-time Updates**: JavaScript integration with API for live data

#### Frontend Architecture
```javascript
// assets/js/app.js - Shared utilities
class MarketMaestroAPI {
    async getPortfolio() {
        const response = await fetch('/api/v1/portfolio');
        return response.json();
    }
    
    async getStrategies() {
        const response = await fetch('/api/v1/strategies');
        return response.json();
    }
}

// Real-time updates via polling
setInterval(async () => {
    const portfolio = await api.getPortfolio();
    updatePortfolioDisplay(portfolio);
}, 5000);
```

## Security & Risk Management

### Risk Management Architecture

#### Risk Parameter Framework
```go
type RiskParameters struct {
    MaxPositionSize      float64 `json:"max_position_size"`
    MaxDailyLoss         float64 `json:"max_daily_loss"`
    MaxDrawdown          float64 `json:"max_drawdown"`
    PositionConcentration float64 `json:"position_concentration"`
    LeverageLimit        float64 `json:"leverage_limit"`
}
```

#### Order Validation Pipeline
```go
func (r *RiskManagerActor) validateOrder(msg ValidateOrderMsg) OrderValidationResponse {
    // 1. Position size validation
    if msg.Quantity > r.getMaxPositionSize(msg.Symbol) {
        return OrderValidationResponse{Approved: false, Reason: "Position size too large"}
    }
    
    // 2. Daily loss limit check
    if r.getDailyLoss() + r.estimateLoss(msg) > r.getMaxDailyLoss() {
        return OrderValidationResponse{Approved: false, Reason: "Daily loss limit exceeded"}
    }
    
    // 3. Portfolio concentration check
    if r.getPositionConcentration(msg.Symbol) > r.getMaxConcentration() {
        return OrderValidationResponse{Approved: false, Reason: "Position concentration too high"}
    }
    
    return OrderValidationResponse{Approved: true}
}
```

#### Risk Metrics Calculation
- **Value at Risk (VaR)**: 95th percentile loss estimation
- **Maximum Drawdown**: Largest peak-to-trough decline
- **Position Concentration**: Percentage of portfolio in single asset
- **Leverage Ratio**: Total exposure relative to account equity

### Security Measures

#### API Key Management
- **Environment Variables**: Sensitive credentials stored in `.env` files
- **Testnet Default**: All exchanges default to testnet for safety
- **Credential Validation**: API keys validated before exchange connection

#### Starlark Sandbox Security
- **Safe Execution**: Starlark provides memory and CPU safety
- **Limited API**: Strategies can only access predefined functions
- **No File System Access**: Strategies cannot read/write files
- **No Network Access**: Strategies cannot make external network calls

#### Actor Isolation
- **Process Isolation**: Each actor runs in isolated goroutines
- **Message Immutability**: All messages are immutable to prevent data races
- **Fault Isolation**: Actor failures don't propagate to other actors
- **Supervision**: Actor supervision handles failures gracefully

## Deployment Architecture

### Single Binary Deployment
```bash
# Build optimized binary
CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o marketmaestro .

# Run with configuration
export BYBIT_TESTNET=false
export LOG_LEVEL=warn
./marketmaestro
```

### Process Management

#### Systemd Service (Recommended)
```ini
[Unit]
Description=MarketMaestro Trading Bot
After=network.target

[Service]
Type=simple
User=marketmaestro
WorkingDirectory=/opt/marketmaestro
ExecStart=/opt/marketmaestro/marketmaestro
Restart=always
RestartSec=10
Environment=BYBIT_TESTNET=false
Environment=LOG_LEVEL=warn

[Install]
WantedBy=multi-user.target
```

#### Docker Deployment
```dockerfile
FROM golang:1.24.4-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o marketmaestro .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/marketmaestro .
COPY --from=builder /app/config.yaml .
CMD ["./marketmaestro"]
```

### Environment Configuration

#### Development Environment
```env
BYBIT_TESTNET=true
BITVAVO_TESTNET=true
LOG_LEVEL=debug
DATABASE_PATH=./dev.db
```

#### Production Environment
```env
BYBIT_TESTNET=false
BITVAVO_TESTNET=false
LOG_LEVEL=warn
DATABASE_PATH=/data/marketmaestro.db
```

### Monitoring and Observability

#### Structured Logging
```go
// Actor logging with context
s.logger.Info().
    Str("actor", "strategy").
    Str("exchange", "bybit").
    Str("symbol", "BTCUSDT").
    Str("strategy", "simple_sma").
    Float64("signal_price", 45000.0).
    Msg("Generated buy signal")
```

#### Health Checks
```go
func (a *APIActor) handleHealth(w http.ResponseWriter, r *http.Request) {
    health := HealthResponse{
        Status:    "healthy",
        Timestamp: time.Now(),
        Version:   BuildVersion,
        Uptime:    time.Since(a.startTime),
        Components: map[string]string{
            "database":  a.checkDatabase(),
            "exchanges": a.checkExchanges(),
            "actors":    a.checkActors(),
        },
    }
    
    json.NewEncoder(w).Encode(health)
}
```

## Development Guidelines

### Actor Development Patterns

#### 1. **Actor Creation**
```go
// Always use engine.Spawn() for actor creation
actorPID := ctx.SpawnChild(func() actor.Receiver {
    return NewMyActor(config, logger)
}, "my_actor")
```

#### 2. **Message Handling**
```go
func (a *MyActor) Receive(ctx *actor.Context) {
    switch msg := ctx.Message().(type) {
    case actor.Started:
        a.onStarted(ctx)
    case MyMessageType:
        a.handleMyMessage(ctx, msg)
    default:
        a.logger.Warn().
            Str("message_type", fmt.Sprintf("%T", msg)).
            Msg("Unknown message received")
    }
}
```

#### 3. **Cross-Actor Communication**
```go
// Send message (fire-and-forget)
ctx.Send(targetPID, MyMessage{Data: "value"})

// Request-response pattern
response, err := ctx.Request(targetPID, MyQuery{}, 5*time.Second)
if err != nil {
    a.logger.Error().Err(err).Msg("Request failed")
    return
}
```

### Error Handling Patterns

#### 1. **Actor Error Handling**
```go
func (a *MyActor) handleRiskyOperation(ctx *actor.Context) {
    defer func() {
        if r := recover(); r != nil {
            a.logger.Error().
                Interface("panic", r).
                Msg("Actor operation panicked")
            // Don't re-panic - log and continue
        }
    }()
    
    // Risky operation here
}
```

#### 2. **Exchange Error Handling**
```go
func (e *ExchangeActor) handleExchangeError(err error) {
    if isTemporaryError(err) {
        e.logger.Warn().Err(err).Msg("Temporary exchange error, will retry")
        e.scheduleReconnect()
    } else {
        e.logger.Error().Err(err).Msg("Permanent exchange error")
        e.disconnect()
    }
}
```

### Testing Patterns

#### 1. **Actor Testing**
```go
func TestMyActor(t *testing.T) {
    // Create test actor system
    engine, err := actor.NewEngine(actor.NewEngineConfig())
    require.NoError(t, err)
    
    // Spawn actor under test
    actorPID := engine.Spawn(func() actor.Receiver {
        return NewMyActor(testConfig, testLogger)
    }, "test_actor")
    
    // Send test message
    response, err := engine.Request(actorPID, TestMessage{}, time.Second)
    require.NoError(t, err)
    
    // Assert response
    assert.Equal(t, expectedResponse, response)
}
```

#### 2. **Strategy Testing**
```go
func TestStrategy(t *testing.T) {
    engine := NewStrategyEngine(testLogger)
    
    // Load test strategy
    err := engine.LoadStrategy("test_strategy.star")
    require.NoError(t, err)
    
    // Test with sample data
    kline := &exchanges.Kline{
        Symbol: "BTCUSDT",
        Close:  45000.0,
        // ... other fields
    }
    
    signal, err := engine.ExecuteOnKline(kline)
    require.NoError(t, err)
    assert.Equal(t, "buy", signal.Action)
}
```

### Performance Considerations

#### 1. **Message Processing**
- Keep message handlers lightweight and fast
- Offload heavy computations to goroutines when necessary
- Use buffered channels for high-frequency data

#### 2. **Memory Management**
- Limit buffer sizes in strategy actors
- Use object pooling for frequently allocated objects
- Monitor goroutine counts and memory usage

#### 3. **Database Operations**
- Use prepared statements for repeated queries
- Batch database operations when possible
- Use database transactions for consistency

### Best Practices

1. **Actor Design**:
   - Keep actors small and focused on single responsibilities
   - Use composition over inheritance for actor capabilities
   - Always handle the `actor.Started` and `actor.Stopped` messages

2. **Message Design**:
   - Use immutable message structures
   - Include all necessary context in messages
   - Follow consistent naming conventions

3. **Error Handling**:
   - Never panic in actor message handlers
   - Use structured logging with appropriate context
   - Implement graceful degradation for non-critical errors

4. **Configuration**:
   - Use environment variables for sensitive data
   - Validate configuration at startup
   - Support configuration hot-reloading where appropriate

5. **Testing**:
   - Test actors in isolation with mock dependencies
   - Use testnet environments for integration testing
   - Implement comprehensive strategy backtesting

This architecture documentation provides a comprehensive overview of MarketMaestro's internal design, enabling developers to understand, maintain, and extend the system effectively.