# Mercantile Strategy Engine Documentation

The Mercantile trading bot uses a powerful **Starlark-based strategy engine** that allows you to write trading strategies in a Python-like scripting language. This document provides comprehensive documentation of all functions, variables, and patterns available to strategy developers.

## Table of Contents
1. [Strategy Engine Overview](#strategy-engine-overview)
2. [Required Strategy Functions](#required-strategy-functions)
3. [Global Variables](#global-variables)
4. [Built-in Functions](#built-in-functions)
5. [Technical Indicators](#technical-indicators)
6. [Utility Functions](#utility-functions)
7. [Signal Return Format](#signal-return-format)
8. [Data Structures](#data-structures)
9. [Best Practices](#best-practices)
10. [Complete Examples](#complete-examples)
11. [Error Handling](#error-handling)

## Strategy Engine Overview

The strategy engine executes Starlark scripts in response to market data events. Each strategy runs in its own actor and receives real-time data through callback functions.

### Key Features
- **Event-driven**: Strategies respond to kline, orderbook, and ticker data
- **Isolated execution**: Each strategy runs in its own sandboxed environment
- **Rich indicator library**: 25+ technical indicators available
- **Flexible configuration**: Strategy-specific settings via `config` object
- **Real-time data**: Access to live market data and historical buffers

### File Structure
```
strategy/
├── simple_sma.star          # Basic moving average strategy
├── rsi_strategy.star         # RSI-based strategy
├── macd_strategy.star        # MACD crossover strategy
└── your_strategy.star        # Your custom strategy
```

## Required Strategy Functions

Every strategy must implement the `settings()` function. Callback functions are optional but at least one should be implemented.

### settings()
**Required**: Yes  
**Purpose**: Define strategy configuration including data interval and parameters

```python
def settings():
    return {
        "interval": "1m",        # Required: Data interval (1m, 5m, 15m, 1h, 4h, 1d)
        "short_period": 10,      # Custom parameter
        "long_period": 20,       # Custom parameter
        "position_size": 0.01    # Custom parameter
    }
```

### on_kline(kline)
**Optional**: Recommended  
**Purpose**: Handle new candlestick data  
**Frequency**: Called on each new kline based on strategy interval

```python
def on_kline(kline):
    """Handle new kline/candlestick data"""
    current_price = kline.close
    
    # Your trading logic here
    if should_buy():
        return {
            "action": "buy",
            "quantity": 0.01,
            "type": "market",
            "reason": "Buy signal triggered"
        }
    
    return {"action": "hold"}
```

### on_orderbook(orderbook)
**Optional**: For high-frequency or spread-based strategies  
**Purpose**: Handle order book updates  
**Frequency**: Called on order book changes

```python
def on_orderbook(orderbook):
    """Handle order book updates"""
    spread = orderbook.asks[0].price - orderbook.bids[0].price
    
    if spread < 0.001:  # Tight spread opportunity
        return {
            "action": "buy", 
            "quantity": 0.01,
            "type": "limit",
            "price": orderbook.bids[0].price + 0.0001,
            "reason": f"Tight spread: {spread}"
        }
    
    return {"action": "hold"}
```

### on_ticker(ticker)
**Optional**: For volume or price movement strategies  
**Purpose**: Handle ticker updates  
**Frequency**: Called on ticker changes

```python
def on_ticker(ticker):
    """Handle ticker updates"""
    if ticker.volume > volume_threshold:
        return {
            "action": "buy",
            "quantity": 0.01, 
            "type": "market",
            "reason": f"High volume: {ticker.volume}"
        }
    
    return {"action": "hold"}
```

## Global Variables

These variables are automatically available in all strategy functions:

### Market Data
- **`symbol`** (string): Current trading pair (e.g., "BTCUSDT")
- **`exchange`** (string): Exchange name (e.g., "bybit", "bitvavo")
- **`close`** (list): Array of closing prices from historical klines
- **`open`** (list): Array of opening prices from historical klines  
- **`high`** (list): Array of high prices from historical klines
- **`low`** (list): Array of low prices from historical klines
- **`volume`** (list): Array of volume data from historical klines
- **`klines`** (list): Full historical kline objects

### Configuration
- **`config`** (dict): Strategy configuration from settings() merged with user overrides

### Order Book Data (when available)
- **`bid`** (float): Best bid price
- **`ask`** (float): Best ask price  
- **`spread`** (float): Current bid-ask spread

### Callback-Specific Variables
- **`kline`** (object): Current kline in `on_kline()` callback
- **`orderbook`** (object): Current orderbook in `on_orderbook()` callback
- **`ticker`** (object): Current ticker in `on_ticker()` callback

## Built-in Functions

### Basic Functions
- **`print(message)`**: Debug output (visible in logs)
- **`len(collection)`**: Get length of lists/strings
- **`range(start, stop, step)`**: Generate number sequences
- **`round(number, precision)`**: Round numbers to specified precision

### Math Functions
- **`math.abs(x)`**: Absolute value

### Logging
- **`log(message)`**: Structured logging (recommended over print)

## Technical Indicators

### Basic Moving Averages

#### Simple Moving Average (SMA)
```python
sma_values = sma(prices, period)
```
- **prices**: List of price values
- **period**: Number of periods (integer)
- **Returns**: List of SMA values

```python
# Example: 20-period SMA of closing prices
sma20 = sma(close, 20)
current_sma = sma20[-1]  # Latest SMA value
```

#### Exponential Moving Average (EMA)
```python
ema_values = ema(prices, period)
```
- **prices**: List of price values  
- **period**: Number of periods (integer)
- **Returns**: List of EMA values

```python
# Example: 12-period EMA
ema12 = ema(close, 12)
```

### Momentum Indicators

#### Relative Strength Index (RSI)
```python
rsi_values = rsi(prices, period)
```
- **prices**: List of price values
- **period**: RSI period (typically 14)
- **Returns**: List of RSI values (0-100)

```python
# Example: Standard 14-period RSI
rsi14 = rsi(close, 14)
current_rsi = rsi14[-1]

if current_rsi > 70:
    # Overbought condition
elif current_rsi < 30:
    # Oversold condition
```

#### MACD (Moving Average Convergence Divergence)
```python
macd_result = macd(prices, fast_period=12, slow_period=26, signal_period=9)
```
- **prices**: List of price values
- **fast_period**: Fast EMA period (default: 12)
- **slow_period**: Slow EMA period (default: 26)  
- **signal_period**: Signal line EMA period (default: 9)
- **Returns**: Dictionary with "macd", "signal", "histogram" arrays

```python
# Example: Standard MACD
macd_data = macd(close)
macd_line = macd_data["macd"]
signal_line = macd_data["signal"]
histogram = macd_data["histogram"]

# Check for bullish crossover
if macd_line[-1] > signal_line[-1] and macd_line[-2] <= signal_line[-2]:
    # MACD crossed above signal line
```

#### Stochastic Oscillator
```python
stoch_result = stochastic(high, low, close, k_period=14, d_period=3)
```
- **high, low, close**: Price arrays
- **k_period**: %K period (default: 14)
- **d_period**: %D period (default: 3)
- **Returns**: Dictionary with "k" and "d" arrays

```python
# Example: Standard Stochastic
stoch = stochastic(high, low, close)
k_values = stoch["k"]
d_values = stoch["d"]
```

### Volatility Indicators

#### Bollinger Bands
```python
bb_result = bollinger(prices, period=20, multiplier=2.0)
```
- **prices**: List of price values
- **period**: Moving average period (default: 20)
- **multiplier**: Standard deviation multiplier (default: 2.0)
- **Returns**: Dictionary with "upper", "middle", "lower" arrays

```python
# Example: Standard Bollinger Bands
bb = bollinger(close, 20, 2.0)
upper_band = bb["upper"]
middle_band = bb["middle"]  # 20-period SMA
lower_band = bb["lower"]

# Check for band squeeze
band_width = (upper_band[-1] - lower_band[-1]) / middle_band[-1]
```

#### Average True Range (ATR)
```python
atr_values = atr(high, low, close, period=14)
```
- **high, low, close**: Price arrays
- **period**: ATR period (default: 14)
- **Returns**: List of ATR values

```python
# Example: 14-period ATR for volatility measurement
atr14 = atr(high, low, close, 14)
current_volatility = atr14[-1]
```

### Volume Indicators

#### Volume Weighted Average Price (VWAP)
```python
vwap_values = vwap(high, low, close, volume)
```
- **high, low, close, volume**: Price and volume arrays
- **Returns**: List of VWAP values

```python
# Example: VWAP calculation
vwap_line = vwap(high, low, close, volume)
current_vwap = vwap_line[-1]

# Price above VWAP suggests bullish sentiment
if close[-1] > current_vwap:
    # Price above VWAP
```

#### Money Flow Index (MFI)
```python
mfi_values = mfi(high, low, close, volume, period=14)
```
- **high, low, close, volume**: Price and volume arrays
- **period**: MFI period (default: 14)
- **Returns**: List of MFI values (0-100)

### Advanced Indicators

#### Average Directional Index (ADX)
```python
adx_result = adx(high, low, close, period=14)
```
- **high, low, close**: Price arrays
- **period**: ADX period
- **Returns**: Dictionary with "adx", "plus_di", "minus_di" arrays

```python
# Example: ADX for trend strength
adx_data = adx(high, low, close, 14)
adx_line = adx_data["adx"]
plus_di = adx_data["plus_di"]
minus_di = adx_data["minus_di"]

current_adx = adx_line[-1]
if current_adx > 25:
    # Strong trend
    if plus_di[-1] > minus_di[-1]:
        # Bullish trend
```

#### Parabolic SAR
```python
psar_values = parabolic_sar(high, low, step=0.02, max_step=0.2)
```
- **high, low**: Price arrays
- **step**: Acceleration factor step (default: 0.02)
- **max_step**: Maximum acceleration factor (default: 0.2)
- **Returns**: List of SAR values

```python
# Example: Parabolic SAR for trend following
psar = parabolic_sar(high, low)
current_price = close[-1]
current_psar = psar[-1]

if current_price > current_psar:
    # Uptrend
else:
    # Downtrend
```

#### Ichimoku Cloud
```python
ichimoku_result = ichimoku(high, low, close, 
                          conversion_period=9, base_period=26, 
                          span_b_period=52, displacement=26)
```
- **Returns**: Dictionary with "tenkan_sen", "kijun_sen", "senkou_span_a", "senkou_span_b", "chikou_span"

```python
# Example: Ichimoku signals
ich = ichimoku(high, low, close)
tenkan = ich["tenkan_sen"][-1]    # Conversion line
kijun = ich["kijun_sen"][-1]      # Base line
senkou_a = ich["senkou_span_a"]   # Leading span A
senkou_b = ich["senkou_span_b"]   # Leading span B

# Bullish signal: Tenkan above Kijun
if tenkan > kijun:
    # Potential uptrend
```

### Support/Resistance Indicators

#### Pivot Points
```python
pivot_result = pivot_points(high, low, close)
```
- **Returns**: Dictionary with "pivot", "r1", "r2", "r3", "s1", "s2", "s3"

```python
# Example: Daily pivot points
pivots = pivot_points(high, low, close)
pivot_level = pivots["pivot"][-1]
resistance1 = pivots["r1"][-1]
support1 = pivots["s1"][-1]
```

#### Fibonacci Retracement
```python
fib_levels = fibonacci(high_price, low_price)
```
- **high_price, low_price**: Single price values (not arrays)
- **Returns**: Dictionary with retracement levels

```python
# Example: Fibonacci retracement between swing high/low
recent_high = max(high[-20:])  # Highest in last 20 periods
recent_low = min(low[-20:])    # Lowest in last 20 periods

fib = fibonacci(recent_high, recent_low)
fib_618 = fib["61.8"]  # Key retracement level
```

### All Available Indicators Summary
- **Basic**: `sma`, `ema`, `rsi`, `stddev`, `roc`
- **Advanced Momentum**: `macd`, `stochastic`, `williams_r`, `cci`, `mfi`
- **Volatility**: `bollinger`, `atr`, `keltner`
- **Volume**: `vwap`, `obv`, `mfi`
- **Trend**: `adx`, `parabolic_sar`, `ichimoku`, `aroon`
- **Support/Resistance**: `pivot_points`, `fibonacci`

## Utility Functions

### Price Analysis
```python
# Rolling highest/lowest values
highest_values = highest(prices, period)
lowest_values = lowest(prices, period)

# Example: Find highest high in last 20 periods
recent_high = highest(high, 20)[-1]
```

### Signal Detection
```python
# Detect when series1 crosses above series2
crossover_signals = crossover(series1, series2)

# Detect when series1 crosses below series2  
crossunder_signals = crossunder(series1, series2)

# Example: MA crossover detection
short_ma = sma(close, 10)
long_ma = sma(close, 20)
bullish_cross = crossover(short_ma, long_ma)

if bullish_cross[-1]:  # Latest value is True
    # Short MA just crossed above Long MA
```

## Signal Return Format

All callback functions should return a dictionary with the following structure:

### Required Fields
- **`action`** (string): "buy", "sell", or "hold"

### Optional Fields
- **`quantity`** (float): Amount to trade (required for buy/sell)
- **`price`** (float): Limit price (for limit orders)
- **`type`** (string): "market" or "limit" (default: "market")
- **`reason`** (string): Human-readable explanation

### Examples

#### Market Buy Order
```python
return {
    "action": "buy",
    "quantity": 0.01,
    "type": "market",
    "reason": "RSI oversold signal"
}
```

#### Limit Sell Order
```python
return {
    "action": "sell", 
    "quantity": 0.01,
    "price": 45000.0,
    "type": "limit",
    "reason": "Take profit at resistance"
}
```

#### Hold Position
```python
return {
    "action": "hold",
    "reason": "Waiting for signal"
}
```

## Data Structures

### Kline Object
```python
kline = {
    "timestamp": "2024-01-01T12:00:00Z",  # ISO timestamp
    "open": 43500.0,                       # Opening price
    "high": 43800.0,                       # Highest price
    "low": 43400.0,                        # Lowest price
    "close": 43750.0,                      # Closing price
    "volume": 125.75                       # Volume traded
}
```

### OrderBook Object
```python
orderbook = {
    "symbol": "BTCUSDT",
    "timestamp": "2024-01-01T12:00:00Z",
    "bids": [                              # Buy orders (price descending)
        {"price": 43749.0, "quantity": 1.5},
        {"price": 43748.0, "quantity": 2.1}
    ],
    "asks": [                              # Sell orders (price ascending)
        {"price": 43751.0, "quantity": 0.8},
        {"price": 43752.0, "quantity": 1.2}
    ]
}
```

### Ticker Object
```python
ticker = {
    "symbol": "BTCUSDT",
    "price": 43750.0,                      # Last traded price
    "volume": 1250.75,                     # 24h volume
    "timestamp": "2024-01-01T12:00:00Z"
}
```

## Best Practices

### 1. Strategy Structure
```python
# Use module-level variables for strategy state
state = {
    "position": 0,
    "last_signal": "hold",
    "indicators": {}
}

# Cache expensive calculations
def update_indicators():
    if len(close) >= 20:
        state["indicators"]["sma20"] = sma(close, 20)
        state["indicators"]["rsi"] = rsi(close, 14)
```

### 2. Data Validation
```python
def on_kline(kline):
    # Always validate data availability
    if len(close) < 20:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Check for valid indicator values
    current_rsi = rsi(close, 14)[-1]
    if math.isnan(current_rsi):
        return {"action": "hold", "reason": "Invalid RSI"}
```

### 3. Risk Management
```python
def calculate_position_size(price, risk_percent=1.0):
    """Calculate position size based on risk percentage"""
    account_balance = 1000.0  # Get from config or API
    risk_amount = account_balance * (risk_percent / 100)
    
    # Calculate stop loss distance
    atr_value = atr(high, low, close, 14)[-1]
    stop_distance = atr_value * 2  # 2 ATR stop
    
    if stop_distance > 0:
        position_size = risk_amount / stop_distance
        return min(position_size, account_balance * 0.1)  # Max 10% of balance
    
    return 0.01  # Default small size
```

### 4. Signal Filtering
```python
def confirm_signal(primary_signal):
    """Use multiple indicators to confirm signals"""
    confirmations = 0
    
    # RSI confirmation
    current_rsi = rsi(close, 14)[-1]
    if primary_signal == "buy" and current_rsi < 50:
        confirmations += 1
    elif primary_signal == "sell" and current_rsi > 50:
        confirmations += 1
    
    # Volume confirmation
    current_volume = volume[-1]
    avg_volume = sma(volume, 20)[-1]
    if current_volume > avg_volume * 1.5:
        confirmations += 1
    
    return confirmations >= 2  # Require at least 2 confirmations
```

## Complete Examples

### Example 1: RSI Mean Reversion Strategy
```python
def settings():
    return {
        "interval": "5m",
        "rsi_period": 14,
        "oversold": 30,
        "overbought": 70,
        "position_size": 0.01
    }

state = {
    "position": 0,
    "entry_price": 0
}

def on_kline(kline):
    # Get configuration
    rsi_period = config.get("rsi_period", 14)
    oversold = config.get("oversold", 30)
    overbought = config.get("overbought", 70)
    position_size = config.get("position_size", 0.01)
    
    # Check data availability
    if len(close) < rsi_period + 1:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Calculate RSI
    rsi_values = rsi(close, rsi_period)
    current_rsi = rsi_values[-1]
    current_price = close[-1]
    
    # Entry signals
    if state["position"] == 0:  # No position
        if current_rsi < oversold:
            state["position"] = 1
            state["entry_price"] = current_price
            return {
                "action": "buy",
                "quantity": position_size,
                "type": "market",
                "reason": f"RSI oversold: {round(current_rsi, 2)}"
            }
    
    # Exit signals
    elif state["position"] == 1:  # Long position
        if current_rsi > overbought or current_price < state["entry_price"] * 0.98:
            state["position"] = 0
            profit = (current_price - state["entry_price"]) / state["entry_price"] * 100
            return {
                "action": "sell",
                "quantity": position_size,
                "type": "market",
                "reason": f"Exit: RSI {round(current_rsi, 2)}, P&L: {round(profit, 2)}%"
            }
    
    return {"action": "hold"}
```

### Example 2: Multi-Indicator Trend Following
```python
def settings():
    return {
        "interval": "15m",
        "fast_ma": 12,
        "slow_ma": 26,
        "rsi_period": 14,
        "atr_period": 14,
        "position_size": 0.02
    }

state = {
    "trend": "none",
    "position": 0
}

def on_kline(kline):
    # Configuration
    fast_period = config.get("fast_ma", 12)
    slow_period = config.get("slow_ma", 26)
    
    # Data validation
    min_periods = max(fast_period, slow_period, 14) + 5
    if len(close) < min_periods:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Calculate indicators
    fast_ma = ema(close, fast_period)
    slow_ma = ema(close, slow_period)
    rsi_values = rsi(close, 14)
    atr_values = atr(high, low, close, 14)
    
    current_price = close[-1]
    current_fast = fast_ma[-1]
    current_slow = slow_ma[-1]
    current_rsi = rsi_values[-1]
    current_atr = atr_values[-1]
    
    # Trend determination
    if current_fast > current_slow * 1.001:  # 0.1% threshold
        state["trend"] = "up"
    elif current_fast < current_slow * 0.999:
        state["trend"] = "down"
    else:
        state["trend"] = "sideways"
    
    # Entry logic
    if state["position"] == 0:
        # Long entry
        if (state["trend"] == "up" and 
            current_rsi > 50 and 
            crossover(fast_ma, slow_ma)[-1]):
            
            state["position"] = 1
            return {
                "action": "buy",
                "quantity": config.get("position_size", 0.02),
                "type": "market",
                "reason": f"Trend: {state['trend']}, RSI: {round(current_rsi, 1)}"
            }
        
        # Short entry
        elif (state["trend"] == "down" and 
              current_rsi < 50 and 
              crossunder(fast_ma, slow_ma)[-1]):
            
            state["position"] = -1
            return {
                "action": "sell",
                "quantity": config.get("position_size", 0.02),
                "type": "market",
                "reason": f"Trend: {state['trend']}, RSI: {round(current_rsi, 1)}"
            }
    
    # Exit logic
    else:
        # Exit long position
        if state["position"] == 1 and crossunder(fast_ma, slow_ma)[-1]:
            state["position"] = 0
            return {
                "action": "sell",
                "quantity": config.get("position_size", 0.02),
                "type": "market",
                "reason": "MA crossunder exit"
            }
        
        # Exit short position
        elif state["position"] == -1 and crossover(fast_ma, slow_ma)[-1]:
            state["position"] = 0
            return {
                "action": "buy",
                "quantity": config.get("position_size", 0.02),
                "type": "market",
                "reason": "MA crossover exit"
            }
    
    return {"action": "hold"}
```

## Error Handling

### Common Issues and Solutions

#### 1. Insufficient Data
```python
# Always check data length before indicator calculations
if len(close) < required_periods:
    return {"action": "hold", "reason": "Insufficient data"}
```

#### 2. NaN Values
```python
import math

indicator_value = rsi(close, 14)[-1]
if math.isnan(indicator_value):
    return {"action": "hold", "reason": "Invalid indicator value"}
```

#### 3. Division by Zero
```python
# Check denominators before division
if denominator != 0:
    ratio = numerator / denominator
else:
    ratio = 0  # or handle appropriately
```

#### 4. Array Index Errors
```python
# Use safe array access
if len(close) > 0:
    current_price = close[-1]
else:
    return {"action": "hold", "reason": "No price data"}
```

### Debug Logging
```python
def on_kline(kline):
    # Use log() for debugging (appears in structured logs)
    log(f"Processing kline: {kline.timestamp}, price: {kline.close}")
    
    # Use print() for simple debugging (appears in stdout)
    print(f"Current position: {state['position']}")
```

---

This documentation covers all functions and capabilities exposed by the Mercantile strategy engine. For additional examples, see the strategy files in the `/strategy` directory.

For questions or issues, refer to the main [README](../README.md) or check the API documentation.