# MarketMaestro Trading Bot - AI Agent Instructions

## Architecture Overview

# MarketMaestro Trading Bot - AI Agent Instructions
Mercantile is a **Hollywood actor framework-based** cryptocurrency trading bot. The entire system is built around message passing between actors, not traditional microservices.

### Core Actor Hierarchy
```
Supervisor (root) → API + UI + Exchange Actors
Exchange Actor → Strategy + OrderManager + RiskManager + Portfolio + Settings
```

**Key Pattern**: All business logic flows through actor message passing. Never bypass actors with direct function calls between components.

## Essential Development Knowledge

### 1. Actor Message Patterns
- **Messages are structs** ending with `Msg`: `StartStrategyMsg`, `KlineDataMsg`, `OrderBookDataMsg`
- **Actor creation**: Use `engine.Spawn()` in Hollywood framework, not direct instantiation
- **Cross-actor communication**: Always via `ctx.Send(pid, message)` or `ctx.Request(pid, message)`
- **State management**: Actors maintain their own state, never share mutable state

Example from `internal/strategy/strategy.go`:
```go
type KlineDataMsg struct{ Kline *exchanges.Kline }

func (s *StrategyActor) Receive(ctx *actor.Context) {
    switch msg := ctx.Message().(type) {
    case KlineDataMsg:
        s.processKlineData(msg.Kline)
    }
}
```

When updating strategy engine functions, technical indicators etc ensure everything is documented in /docs/strategy-engine.md including examples and usage patterns. Ensure all technical indicators have unit tests to validate functionality.

### 2. Exchange Interface Pattern
- **Abstract interface**: `pkg/exchanges/interface.go` defines common exchange operations
- **Factory pattern**: `pkg/exchanges/factory.go` creates exchange instances by name
- **Implementation**: Each exchange (Bybit, Bitvavo) implements the interface separately
- **Configuration**: Exchange-specific settings in `config.yaml` under `exchanges.{name}`

### 3. Starlark Strategy Engine
- **Strategy scripts**: Written in Starlark (Python-like) in `/strategy/*.star` files
- **Required functions**: `settings()`, `on_kline()`, `on_orderbook()`, `on_ticker()`
- **Interval extraction**: Strategies define their own `interval` in `settings()` return value
- **State persistence**: Use `state` dict in Starlark for strategy-specific data

Example pattern from `strategy/simple_sma.star`:
```python
def settings():
    return {"interval": "1m", "short_period": 10}

def on_kline(kline):
    # Strategy logic here
    if should_buy():
        signal("buy", 0.01)
```

### 4. Configuration Management
- **Dual config**: Environment variables (`.env`) + YAML file (`config.yaml`)
- **Environment**: API keys, ports, log levels
- **YAML**: Strategy configs, exchange pairs, risk parameters
- **Loading**: Use `pkg/config/Load()` which merges both sources

### 5. Database & Migrations
- **SQLite**: Single file database with actor-based state persistence
- **Migrations**: Auto-run on startup via `pkg/database/migrations/`
- **Actor persistence**: Settings table stores actor configuration as key-value pairs
- **Trading data**: Orders, positions, portfolio snapshots in dedicated tables

## Critical Development Workflows

### Build & Run
```bash
go build -o bin/mercantile .    # Build binary
./bin/mercantile                # Run with .env + config.yaml
```

### Access Points
- **Web UI**: http://localhost:8081 (embedded assets, no external build)
- **API**: http://localhost:8080/api/v1/ (Chi router with OpenAPI)
- **Health**: http://localhost:8080/api/v1/health

### Adding New Strategies
1. Create `strategy/{name}.star` with required functions
2. Add strategy config to `config.yaml` under exchange pairs
3. Strategy actor auto-spawns based on configuration

### UI Development
- **Embedded assets**: Use `//go:embed assets/*` pattern in `internal/ui/ui.go`
- **Template system**: Go templates with shared base template and blocks
- **Shared components**: CSS/JS components in `/assets/` for consistency
- **API integration**: Frontend uses shared JavaScript utils in `assets/js/app.js`

## Project-Specific Conventions

### Error Handling
- **Actor errors**: Log with structured zerolog, don't panic
- **HTTP errors**: Return proper status codes, use Chi middleware
- **Exchange errors**: Wrap in custom error types for retry logic

### Logging
- **Structured logging**: Use zerolog with actor context
- **Actor identification**: Always include `actor`, `exchange`, `strategy` fields
- **Message tracing**: Log message types for debugging actor communication

### Testing Exchange Integrations
- **Testnet first**: All exchanges default to testnet in development
- **Mock data**: Use `test/` directory for integration tests with real API responses
- **WebSocket handling**: Exchanges connect via WebSocket for real-time data

### File Organization
- **Internal packages**: Business logic in `/internal/`
- **Shared packages**: Reusable code in `/pkg/`
- **Strategy scripts**: Starlark files in `/strategy/`
- **Static assets**: UI assets embedded in `/internal/ui/assets/`

## Integration Points

### Exchange → Strategy Flow
1. Exchange actor subscribes to WebSocket feeds
2. Receives kline/orderbook data → forwards as `KlineDataMsg` to strategy actors
3. Strategy processes data → sends `OrderRequestMsg` back to exchange
4. Exchange validates via risk manager → executes via order manager

### API ↔ Actor Communication
- **API actor**: Bridge between REST endpoints and actor system
- **Status queries**: API requests status from actors via message passing
- **Real-time data**: WebSocket connections stream actor state to frontend

This architecture requires thinking in **message passing** and **actor isolation** rather than traditional function calls and shared state.

# Changelog

keep a CHANGELOG.md in the root of the project to track changes, improvements, and bug fixes. This helps maintain clarity on what has been modified over time.

# Mock functions

Do not implement mock functions always use live data or testnet environments for testing. Mocking can lead to discrepancies between test and production behavior, especially in trading systems where real-time data is crucial for decision-making.

# API

Ensure all API endpoints are documented in the OpenAPI specification. When adding new features to the bot also expose an API for the features. The api endpoint should be fully functional and never use mock data.