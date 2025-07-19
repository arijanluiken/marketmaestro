# Enhanced Order Management System

## Overview

The enhanced order management system provides advanced trading functionality including trailing stops, stop-limit orders, and comprehensive interaction between strategy, order manager, and exchange actors.

## Architecture

### 1. Order Manager Actor (`internal/order/order.go`)

**Enhanced Order Types:**
- `OrderTypeMarket` - Immediate execution at market price
- `OrderTypeLimit` - Execution at specified price or better
- `OrderTypeStopMarket` - Market order triggered at stop price
- `OrderTypeStopLimit` - Limit order triggered at stop price
- `OrderTypeTrailing` - Trailing stop with dynamic adjustment

**Key Features:**
- **Advanced Order Management**: Supports trailing stops, stop-loss orders
- **Real-time Price Monitoring**: Tracks market prices for stop order triggers
- **Thread-safe Operations**: Uses mutex for concurrent access
- **Order State Persistence**: Enhanced order tracking with metadata

**Message Types:**
```go
PlaceOrderMsg          // Standard order placement
PlaceTrailingStopMsg   // Trailing stop orders
PlaceStopOrderMsg      // Stop market/limit orders
ModifyOrderMsg         // Order modification
PriceUpdateMsg         // Real-time price updates
```

### 2. Strategy Actor (`internal/strategy/strategy.go`)

**Callback-based Execution:**
- `executeKlineCallback()` - Process price/volume data
- `executeOrderBookCallback()` - Process order book depth
- `executeTickerCallback()` - Process ticker price updates

**Enhanced Features:**
- **Price Relay**: Forwards market data to order manager
- **Signal Processing**: Converts strategy signals to order requests
- **Risk Integration**: Coordinates with risk manager

### 3. Exchange Actor (`internal/exchange/exchange.go`)

**Data Broadcasting:**
- Routes market data to strategies and order manager
- Calculates mid-price from order book for stop orders
- Provides real-time price feeds for trailing stops

## Advanced Order Types

### 1. Trailing Stop Orders

**How it works:**
- Tracks the highest price (for sell) or lowest price (for buy)
- Adjusts stop price based on percentage or absolute amount
- Triggers market order when price reverses beyond trail distance

**Example Usage in Strategy:**
```python
return {
    "action": "buy",
    "quantity": 0.01,
    "type": "market",
    "trail_percent": 2.0,  # 2% trailing stop
    "reason": "Bullish crossover with trailing stop"
}
```

**Implementation:**
- Order manager monitors price updates
- Updates high water mark continuously
- Triggers when price reverses beyond trail threshold

### 2. Stop Market/Stop Limit Orders

**How it works:**
- Waits for price to reach trigger level
- Converts to market or limit order upon trigger
- Provides downside protection and upside entry

**Example Usage:**
```python
return {
    "action": "sell",
    "quantity": 0.01,
    "type": "stop_market",
    "stop_price": 50000.0,
    "reason": "Stop loss protection"
}
```

### 3. Enhanced Position Management

**Features:**
- Dynamic stop loss adjustment
- Take profit automation
- Position size management
- Risk-based order sizing

## Message Flow

### Order Placement Flow:
1. **Strategy** generates signal with enhanced parameters
2. **Strategy Actor** processes signal and forwards to **Order Manager**
3. **Order Manager** determines order type and execution method:
   - Regular orders → Send to **Exchange**
   - Advanced orders → Store and monitor
4. **Exchange** executes regular orders and reports back

### Price Update Flow:
1. **Exchange** receives market data (klines, orderbook, ticker)
2. **Exchange** calculates relevant prices (close, mid-price)
3. **Exchange** broadcasts to:
   - **Strategy Actors** (for callback execution)
   - **Order Manager** (for stop order monitoring)
   - **Portfolio** (for P&L calculation)

### Stop Order Execution Flow:
1. **Order Manager** receives price update
2. Checks all pending stop/trailing orders
3. If trigger condition met:
   - Creates market/limit order
   - Sends to **Exchange** for execution
   - Updates order status and persistence

## Strategy Integration

### Enhanced Strategy Script (`strategy/enhanced_sma.star`)

**Features:**
- Callback-based execution (on_kline, on_orderbook, on_ticker)
- Advanced order type support
- Position management logic
- Risk-based parameters

**Order Signal Enhancement:**
```python
def check_signals(closes):
    # ... MA calculation ...
    
    if bullish_crossover:
        return {
            "action": "buy",
            "quantity": position_size,
            "type": "market",
            "trail_percent": 2.0,      # Trailing stop
            "stop_price": stop_loss,   # Fixed stop
            "take_profit": tp_price,   # Take profit level
            "reason": "Bullish MA crossover"
        }
```

## Configuration

### Strategy Settings:
```python
def settings():
    return {
        "interval": "1m",
        "use_trailing_stops": True,
        "trail_percent": 2.0,
        "stop_loss_percent": 1.5,
        "take_profit_percent": 3.0
    }
```

### Order Manager Configuration:
- Price monitoring frequency: 1 second
- Thread-safe concurrent operations
- Automatic order persistence
- Real-time trigger evaluation

## Benefits

1. **Risk Management**: Automated stop losses and trailing stops
2. **Efficiency**: Real-time order monitoring without manual intervention
3. **Flexibility**: Multiple order types for different market conditions
4. **Scalability**: Actor-based architecture handles multiple symbols/strategies
5. **Reliability**: Thread-safe operations and state persistence

## Usage Examples

### Basic Trailing Stop:
```python
# In strategy script
return {
    "action": "buy",
    "quantity": 0.01,
    "type": "market",
    "trail_percent": 2.0,
    "reason": "Entry with 2% trailing stop"
}
```

### Stop Loss with Take Profit:
```python
# In strategy script  
return {
    "action": "buy",
    "quantity": 0.01,
    "type": "market",
    "stop_price": current_price * 0.98,     # 2% stop loss
    "take_profit": current_price * 1.05,    # 5% take profit
    "reason": "Entry with fixed stops"
}
```

### Dynamic Position Management:
```python
# Position adjustment based on P&L
if profit_percent > 2.0:
    return {
        "action": "hold",
        "trail_percent": 1.0,  # Tighten trailing stop
        "reason": "Tightening stop in profit"
    }
```

This enhanced order management system provides institutional-grade trading capabilities with automated risk management, advanced order types, and real-time market monitoring.
