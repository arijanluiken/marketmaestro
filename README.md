# Mercantile Trading Bot

An advanced cryptocurrency trading bot built in Go using the actor model with the Hollywood framework.

## Features

- **Actor-Based Architecture**: Clean separation of concerns using the Hollywood actor framework
- **Multi-Exchange Support**: Abstract interface with implementations for Bybit and Bitvavo
- **REST API**: Full-featured API with OpenAPI specification
- **Web Interface**: Modern, responsive UI built with Pure CSS
- **Strategy Engine**: Designed for Starlark-based trading strategies
- **Risk Management**: Built-in risk management and portfolio tracking
- **Database Integration**: SQLite with automated migrations
- **Configuration Management**: Environment variables and YAML configuration

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd mercantile
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your exchange API credentials
   ```

3. **Build and run**
   ```bash
   go build -o bin/mercantile .
   ./bin/mercantile
   ```

4. **Access the application**
   - Web UI: http://localhost:8081
   - API: http://localhost:8080/api/v1/health
   - OpenAPI Spec: http://localhost:8080/api/v1/openapi.json

## Architecture

### Actor Hierarchy
- **Supervisor Actor**: Root actor managing the entire system
  - **API Actor**: REST API server
  - **UI Actor**: Web interface server
  - **Exchange Actors**: One per configured exchange
    - **Strategy Actors**: Trading strategy execution
    - **Order Manager Actor**: Order placement and management
    - **Risk Manager Actor**: Risk assessment and limits
    - **Portfolio Actor**: Balance and P&L tracking
    - **Settings Actor**: Configuration persistence

### Configuration

The bot uses a combination of environment variables (`.env`) and YAML configuration (`config.yaml`):

**Environment Variables (.env):**
```env
BYBIT_API_KEY=your_api_key
BYBIT_SECRET=your_secret
BYBIT_TESTNET=true

BITVAVO_API_KEY=your_api_key
BITVAVO_SECRET=your_secret
BITVAVO_TESTNET=true

API_PORT=8080
UI_PORT=8081
LOG_LEVEL=info
```

**YAML Configuration (config.yaml):**
```yaml
database:
  path: "./mercantile.db"

api:
  port: 8080
  timeout: 30s

ui:
  port: 8081

logging:
  level: "info"
```

## Development Status

✅ **Completed:**
- Core actor system with message passing
- Exchange interface and factory pattern
- REST API with Chi router
- Web UI with embedded assets
- Database schema and migrations
- Configuration management
- Structured logging

🚧 **In Progress:**
- WebSocket data feeds
- Starlark strategy engine
- Complete exchange implementations
- Advanced order types
- Real-time portfolio tracking

## API Endpoints

- `GET /api/v1/health` - Health check
- `GET /api/v1/openapi.json` - OpenAPI specification
- `GET /api/v1/exchanges` - List configured exchanges
- `GET /api/v1/strategies` - List trading strategies
- More endpoints documented in the OpenAPI spec

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.