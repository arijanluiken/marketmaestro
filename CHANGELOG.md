# Changelog

All notable changes to the Mercantile Trading Bot project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Order Book Data Handling**: Improved processing of partial order books during low liquidity periods
  - Exchange actors now only warn for completely empty order books (0 bids AND 0 asks)
  - Partial order books (bids only or asks only) are logged at debug level instead of warning
  - Strategy actors gracefully handle partial order books with appropriate price calculation
  - Order managers receive price updates using best available price (bid, ask, or mid-price)
  - Reduced noisy warnings that were previously logged for normal market conditions

- **Chart Price Accuracy**: Fixed BTCUSDT charts showing incorrect prices (~$128k) by implementing USD index price adjustment for Bybit testnet. Charts now display real-world Bitcoin prices (~$118k) while preserving OHLC candlestick patterns and volume data. The fix applies a dynamic ratio based on the difference between Bybit's spot market price and USD index price to ensure accurate price representation.

### Improved  
- **Price Data Quality**: Enhanced Bybit exchange integration to use USD index price for testnet environments, providing more accurate pricing that matches real market values
- **Chart Visualization**: Improved candlestick chart rendering of "flat" candles (where OHLC values are nearly identical) to make them more visible with a minimum body height
- **Logging**: Added detailed logging for USD index price retrieval and ratio calculations to help debug pricing issues

### Added
- **Structured Logging**: Migrated main.go from standard Go log to zerolog
  - Enhanced console output with timestamps and log levels
  - Structured logging format consistent with actor system
  - Better error visibility and debugging capabilities

- **Database Migration System Fixes**: Resolved migration conflicts and dirty state handling
  - Fixed conflicting settings table definitions between migrations 001 and 003
  - Added automatic dirty state cleanup for interrupted migrations
  - Restructured migration 001 to remove settings table creation
  - Updated migration 003 to properly drop and recreate settings table with exchange-based schema
  - Enhanced migration error handling with force clean capabilities

- **Rebalance Actor System**: Complete portfolio rebalancing system with Starlark scripting
  - Rebalance actor as child of exchange actor
  - Starlark-based rebalancing scripts in `/rebalance/` directory
  - `on_rebalance()` callback function support
  - Exposed functions: `get_balances()`, `get_current_prices()`, `place_order()`, `log()`
  - REST API endpoints for rebalance control (`/exchanges/{exchange}/rebalance/*`)
  - Example equal weight rebalancing strategy
  
- **Enhanced Technical Indicators**: 10 new advanced technical indicators
  - **ZigZag**: Trend reversal detection with customizable deviation threshold
  - **Percentile Rank**: Relative strength analysis (0-100%)
  - **Linear Regression**: Trend direction and slope analysis with R-squared
  - **Kaufman Efficiency Ratio**: Market efficiency measurement (0-1)
  - **Mass Index**: Reversal identification using volatility analysis
  - **Coppock Curve**: Long-term trend identification indicator
  - **Weighted Moving Average (WMA)**: Price-weighted averaging method
  - **Choppiness Index**: Sideways vs trending market detection
  - **Standard Error**: Trend reliability measurement for regression
  - **R-Squared**: Trend strength correlation coefficient (0-1)

- **Strategy Engine Improvements**:
  - Complete Starlark integration for all new indicators
  - Comprehensive error handling and input validation
  - NaN handling for insufficient data periods
  - Type safety with proper Starlark conversions
  - Advanced demo strategies showcasing new indicators

- **Documentation Updates**:
  - Updated strategy engine documentation with new indicators
  - Complete usage examples and parameter descriptions
  - New indicator calculation methods and best practices
  - Advanced strategy patterns and market analysis techniques

- **Testing Infrastructure**:
  - Comprehensive test suite for new indicators
  - Validation of calculation accuracy and edge cases
  - Integration tests for Starlark wrapper functions
  - Performance testing for indicator calculations

### Enhanced
- **Order Management System**: Improved integration between actors
  - Better message passing between order manager, risk manager, and settings actors
  - Enhanced error handling and validation
  - Improved actor reference management

- **Exchange Actor Architecture**: 
  - Enhanced child actor spawning and management
  - Improved message forwarding between exchange and child actors
  - Better actor lifecycle management

- **API System**:
  - New rebalance management endpoints
  - Improved error handling and response formatting
  - Better integration with actor message passing

### Fixed
- **Build System**: Resolved compilation issues and import dependencies
- **Actor Communication**: Fixed message passing patterns and actor references
- **Type Safety**: Corrected Starlark API usage and type conversions
- **File Management**: Cleaned up empty and unused files

### Technical Improvements
- **Architecture**: Hollywood actor framework-based message passing
- **Scripting**: Starlark (Python-like) scripting for strategies and rebalancing
- **Real-time Data**: WebSocket integration for live market data
- **Database**: SQLite with automatic migrations
- **Configuration**: Dual config system (environment variables + YAML)

### Repository Cleanup
- Removed redundant markdown documentation files from root directory
- Consolidated documentation into `/docs/` folder
- Removed test directory from root (moved to proper location)
- Improved project structure and organization

---

## [Previous Versions]

### [0.1.0] - Initial Release
- Basic trading bot framework
- Simple moving average strategies
- Exchange integrations (Bybit, Bitvavo)
- Basic actor system architecture
- Web UI for monitoring
- REST API for control