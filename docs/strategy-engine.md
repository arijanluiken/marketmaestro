# MarketMaestro Strategy Engine Documentation

The MarketMaestro trading bot uses a powerful **Starlark-based strategy engine** that allows you to write trading strategies in a Python-like scripting language. This document provides comprehensive documentation of all functions, variables, and patterns available to strategy developers.

## Table of Contents
1. [Strategy Engine Overview](#strategy-engine-overview)
2. [Required Strategy Functions](#required-strategy-functions)
3. [Thread-Safe Architecture](#thread-safe-architecture)
4. [Configuration Management](#configuration-management)
5. [State Management](#state-management)
6. [Built-in Functions](#built-in-functions)
7. [Technical Indicators](#technical-indicators)
8. [Utility Functions](#utility-functions)
9. [Signal Return Format](#signal-return-format)
10. [Data Structures](#data-structures)
11. [Best Practices](#best-practices)
12. [Complete Examples](#complete-examples)
13. [Error Handling](#error-handling)

## Strategy Engine Overview

The strategy engine executes Starlark scripts in response to market data events. Each strategy runs in its own actor and receives real-time data through callback functions using a **thread-safe architecture**.

### Key Features
- **Event-driven**: Strategies respond to kline, orderbook, and ticker data
- **Isolated execution**: Each strategy runs in its own sandboxed environment
- **Thread-safe state**: Uses `get_state()` and `set_state()` for thread-safe state management
- **Runtime configuration**: Dynamic config access with `get_config()` and fallback defaults
- **Rich indicator library**: 25+ technical indicators available
- **Flexible configuration**: Strategy-specific settings with user overrides
- **Real-time data**: Access to live market data and historical buffers

### File Structure
```
strategy/
├── simple_sma.star          # Basic moving average strategy
├── rsi_strategy.star         # RSI-based strategy
├── macd_strategy.star        # MACD crossover strategy
├── enhanced_sma.star         # Advanced SMA with order management
├── multi_indicator_strategy.star  # Multi-indicator analysis
└── your_strategy.star        # Your custom strategy
```

## Required Strategy Functions

Every strategy must implement the `settings()` function and helper functions for the new thread-safe architecture. Callback functions are optional but at least one should be implemented.

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

### get_config_values()
**Required**: Yes (for thread-safe config access)  
**Purpose**: Access runtime configuration with fallback defaults

```python
def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    params = settings()  # Get default values
    return {
        "short_period": get_config("short_period", params["short_period"]),
        "long_period": get_config("long_period", params["long_period"]),
        "position_size": get_config("position_size", params["position_size"])
    }
```

### init_state()
**Required**: Yes (for thread-safe state management)  
**Purpose**: Initialize strategy state variables using thread-safe methods

```python
def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("short_ma", [])
        set_state("long_ma", [])
        set_state("last_signal", "hold")
```

### on_kline(kline)
**Optional**: Recommended  
**Purpose**: Handle new candlestick data  
**Frequency**: Called on each new kline based on strategy interval

```python
def on_kline(kline):
    """Handle new kline/candlestick data"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    
    # Add new kline to buffer
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only needed klines
    max_needed = cfg["long_period"] + 10
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Your trading logic here
    current_price = kline.close
    if should_buy(cfg, klines):
        return {
            "action": "buy",
            "quantity": cfg["position_size"],
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

### on_start()
**Optional**: For strategy initialization  
**Purpose**: Initialize strategy state, validate configuration, setup variables  
**Frequency**: Called once when strategy starts

```python
def on_start():
    """Initialize strategy when it starts"""
    print("🚀 Strategy starting")
    print("📊 Initializing technical analysis parameters")
    
    # Initialize state
    init_state()
    
    # Get and validate configuration
    cfg = get_config_values()
    if cfg.get("risk_percent", 0) > 5:
        print("⚠️  High risk percentage detected")
    
    print("✅ Strategy initialization complete")
```

### on_stop()
**Optional**: For strategy cleanup  
**Purpose**: Clean up resources, save final state, log summary  
**Frequency**: Called once when strategy stops

```python
def on_stop():
    """Clean up when strategy stops"""
    print("🛑 Strategy stopping")
    print("💾 Saving final state")
    
    # Log strategy performance summary
    # Close any open positions if needed
    # Clean up resources
    
    print("✅ Strategy stopped cleanly")
```

## Thread-Safe Architecture

The strategy engine uses a thread-safe architecture to prevent race conditions and state corruption. **Never use global variables for state management**.

### Key Principles

1. **Use `get_state()` and `set_state()` for all state variables**
2. **Use `get_config()` for runtime configuration access**
3. **Initialize state in `init_state()` function**
4. **Call `init_state()` at the beginning of each callback**

### Thread-Safe State Management

```python
# ✅ CORRECT: Thread-safe state management
def on_kline(kline):
    # Always initialize state first
    init_state()
    
    # Get current state safely
    klines = get_state("klines", [])
    last_signal = get_state("last_signal", "hold")
    
    # Update state safely
    klines.append(new_kline_data)
    set_state("klines", klines)
    set_state("last_signal", "buy")

# ❌ WRONG: Global state (causes frozen state errors)
state = {"klines": [], "last_signal": "hold"}  # Don't do this!

def on_kline(kline):
    state["klines"].append(kline)  # This will fail!
```

### State Management Best Practices

1. **Always call `init_state()` at the start of callbacks**
2. **Use descriptive state keys**: `"rsi_values"`, `"moving_averages"`, `"position_size"`
3. **Provide defaults**: `get_state("klines", [])` instead of `get_state("klines")`
4. **Update state atomically**: Get → Modify → Set

```python
def init_state():
    """Initialize all state variables"""
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("indicators", {})
        set_state("position", 0)
        set_state("entry_price", 0.0)
        set_state("last_signal", "hold")
```

## Configuration Management

The new configuration system supports default values with user overrides and runtime access.

### Configuration Flow

1. **Strategy defaults** defined in `settings()` function
2. **User overrides** from `config.yaml` (per-strategy, per-symbol)
3. **Runtime access** via `get_config()` with fallbacks
4. **Thread-safe access** in all callback functions

### Configuration Pattern

```python
# 1. Define defaults in settings()
def settings():
    return {
        "interval": "15m",  # Can be overridden in config.yaml
        "period": 14,
        "oversold": 30,
        "overbought": 70,
        "position_size": 0.01
    }

# 2. Create config access function
def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    params = settings()
    return {
        "period": get_config("period", params["period"]),
        "oversold": get_config("oversold", params["oversold"]),
        "overbought": get_config("overbought", params["overbought"]),
        "position_size": get_config("position_size", params["position_size"])
    }

# 3. Use in callbacks
def on_kline(kline):
    init_state()
    cfg = get_config_values()  # Get runtime config
    
    # Use config values
    if len(closes) >= cfg["period"]:
        rsi_values = rsi(closes, cfg["period"])
        
        if rsi_values[-1] < cfg["oversold"]:
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
                "reason": f"RSI oversold: {rsi_values[-1]}"
            }
```

### User Configuration Override

In `config.yaml`, users can override strategy parameters:

```yaml
strategy_defaults:
  interval: "5m"          # Global default
  position_size: 0.01     # Global default

exchanges:
  bybit:
    strategies:
      BTCUSDT:
        rsi_strategy:
          interval: "15m"    # Override for RSI strategy on BTCUSDT
          period: 21         # Custom RSI period
          oversold: 25       # Custom oversold level
      ETHUSDT:
        simple_sma:
          interval: "5m"     # Different interval for ETH
          short_period: 5    # Faster SMA periods
          long_period: 15
```

## Built-in Functions

### Thread-Safe State Management
- **`get_state(key, default=None)`**: Get state value safely with optional default
- **`set_state(key, value)`**: Set state value safely (thread-safe)

```python
# Thread-safe state access
def on_kline(kline):
    # Get state with default fallback
    klines = get_state("klines", [])
    position = get_state("position", 0)
    
    # Update state safely
    klines.append(new_kline)
    set_state("klines", klines)
    set_state("position", 1)
```

### Configuration Access
- **`get_config(key, default=None)`**: Get configuration value with fallback

```python
# Configuration access with defaults
def get_config_values():
    return {
        "period": get_config("period", 14),        # Default to 14 if not configured
        "threshold": get_config("threshold", 0.5)  # Default to 0.5 if not configured
    }
```

### Basic Functions
- **`print(message)`**: Debug output (visible in logs)
- **`len(collection)`**: Get length of lists/strings
- **`range(start, stop, step)`**: Generate number sequences
- **`round(number, precision)`**: Round numbers to specified precision

### Math Functions
- **`math.abs(x)`**: Absolute value
- **`math.isnan(x)`**: Check if value is NaN

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
# Example: 20-period SMA using thread-safe state management
def on_kline(kline):
    # Initialize state
    init_state()
    
    # Get current state
    klines = get_state("klines", [])
    
    # Add new kline
    klines.append({
        "timestamp": kline.timestamp,
        "close": kline.close,
        "high": kline.high,
        "low": kline.low,
        "volume": kline.volume
    })
    
    # Update state
    set_state("klines", klines)
    
    # Extract close prices 
    close_prices = [k["close"] for k in klines]
    
    if len(close_prices) >= 20:
        sma20 = sma(close_prices, 20)
        current_sma = sma20[-1]  # Latest SMA value
        
        # Store SMA in state for future reference
        set_state("sma20", sma20)
```

#### Exponential Moving Average (EMA)
```python
ema_values = ema(prices, period)
```
- **prices**: List of price values  
- **period**: Number of periods (integer)
- **Returns**: List of EMA values

```python
# Example: 12-period EMA using thread-safe pattern
def on_kline(kline):
    init_state()
    cfg = get_config_values()
    
    # Get current klines from state
    klines = get_state("klines", [])
    close_prices = [k["close"] for k in klines]
    
    if len(close_prices) >= cfg["ema_period"]:
        ema12 = ema(close_prices, cfg["ema_period"])
        set_state("ema12", ema12)  # Store for later use
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
# Example: Standard 14-period RSI using thread-safe state
def on_kline(kline):
    init_state()
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    close_prices = [k["close"] for k in klines]
    
    if len(close_prices) >= cfg["rsi_period"]:
        rsi_values = rsi(close_prices, cfg["rsi_period"])
        current_rsi = rsi_values[-1]
        
        # Store RSI values in state
        set_state("rsi_values", rsi_values)
        
        # Check trading conditions
        if current_rsi > cfg["overbought"]:
            set_state("last_signal", "sell")
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "reason": f"RSI overbought: {current_rsi}"
            }
        elif current_rsi < cfg["oversold"]:
            set_state("last_signal", "buy")
            return {
                "action": "buy", 
                "quantity": cfg["position_size"],
                "reason": f"RSI oversold: {current_rsi}"
            }
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
# Example: Standard MACD using thread-safe pattern
def on_kline(kline):
    init_state()
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    close_prices = [k["close"] for k in klines]
    
    if len(close_prices) >= cfg["macd_slow_period"] + cfg["macd_signal_period"]:
        macd_data = macd(close_prices, cfg["macd_fast"], cfg["macd_slow"], cfg["macd_signal"])
        macd_line = macd_data["macd"]
        signal_line = macd_data["signal"]
        histogram = macd_data["histogram"]
        
        # Store MACD data in state
        set_state("macd_line", macd_line)
        set_state("signal_line", signal_line)
        set_state("histogram", histogram)
        
        # Check for bullish crossover
        if len(macd_line) >= 2 and len(signal_line) >= 2:
            if macd_line[-1] > signal_line[-1] and macd_line[-2] <= signal_line[-2]:
                # MACD crossed above signal line
                return {
                    "action": "buy",
                    "quantity": cfg["position_size"],
                    "reason": "MACD bullish crossover"
                }
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
# Extract from internal buffer in callback function
def on_kline(kline):
    close_prices = [k["close"] for k in state["klines"]]
    
    if len(close_prices) >= 20:
        bb = bollinger(close_prices, 20, 2.0)
        upper_band = bb["upper"]
        middle_band = bb["middle"]  # 20-period SMA
        lower_band = bb["lower"]
        
        # Check for band squeeze
        if len(upper_band) > 0 and len(middle_band) > 0 and len(lower_band) > 0:
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
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    close_prices = [k["close"] for k in state["klines"]]
    
    if len(close_prices) >= 14:
        atr14 = atr(high_prices, low_prices, close_prices, 14)
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
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    close_prices = [k["close"] for k in state["klines"]]
    volume_data = [k["volume"] for k in state["klines"]]
    
    if len(close_prices) >= 20:
        vwap_line = vwap(high_prices, low_prices, close_prices, volume_data)
        current_vwap = vwap_line[-1]
        current_price = kline.close
        
        # Price above VWAP suggests bullish sentiment
        if current_price > current_vwap:
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
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    
    if len(high_prices) >= 20:
        psar = parabolic_sar(high_prices, low_prices)
        current_price = kline.close
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
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    
    if len(high_prices) >= 20:
        recent_high = max(high_prices[-20:])  # Highest in last 20 periods
        recent_low = min(low_prices[-20:])    # Lowest in last 20 periods
        
        fib = fibonacci(recent_high, recent_low)
        fib_618 = fib["61.8"]  # Key retracement level
```

#### True Strength Index (TSI)
```python
tsi_values = tsi(prices, long_period=25, short_period=13)
```
- **prices**: Price array (typically close prices)
- **long_period**: Long smoothing period (default: 25)
- **short_period**: Short smoothing period (default: 13)
- **Returns**: List of TSI values (-100 to 100)

```python
# Example: TSI momentum analysis
tsi_line = tsi(close, 25, 13)
current_tsi = tsi_line[-1]

if current_tsi > 25:
    # Strong bullish momentum
elif current_tsi < -25:
    # Strong bearish momentum
```

#### Donchian Channels
```python
donchian_result = donchian(high, low, period=20)
```
- **high, low**: Price arrays
- **period**: Channel period (default: 20)
- **Returns**: Dictionary with "upper", "middle", "lower" channel lines

```python
# Example: Donchian breakout strategy
# Extract from internal buffer in callback function  
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    
    if len(high_prices) >= 20:
        channels = donchian(high_prices, low_prices, 20)
        upper_channel = channels["upper"][-1]
        lower_channel = channels["lower"][-1]
        current_price = kline.close
        
        if current_price > upper_channel:
            # Upside breakout
        elif current_price < lower_channel:
            # Downside breakout
```

#### Advanced CCI with Smoothing
```python
advanced_cci_values = advanced_cci(high, low, close, period=14, smooth_period=3)
```
- **high, low, close**: Price arrays
- **period**: CCI calculation period
- **smooth_period**: Additional smoothing period
- **Returns**: List of smoothed CCI values

```python
# Example: Advanced CCI for divergence analysis
cci_smooth = advanced_cci(high, low, close, 14, 3)
current_cci = cci_smooth[-1]

if current_cci > 100:
    # Overbought condition
elif current_cci < -100:
    # Oversold condition
```

#### Elder Ray Index
```python
elder_result = elder_ray(high, low, close, period=13)
```
- **high, low, close**: Price arrays
- **period**: EMA period for calculations
- **Returns**: Dictionary with "bull_power" and "bear_power" arrays

```python
# Example: Elder Ray bullish/bearish power
elder = elder_ray(high, low, close, 13)
bull_power = elder["bull_power"][-1]
bear_power = elder["bear_power"][-1]

if bull_power > 0 and bear_power > bear_power[-2]:
    # Bulls gaining strength
elif bear_power < 0 and bull_power < bull_power[-2]:
    # Bears gaining strength
```

#### Detrended Price Oscillator (DPO)
```python
dpo_values = detrended(prices, period=14)
```
- **prices**: Price array
- **period**: Detrending period
- **Returns**: List of detrended values

```python
# Example: DPO for cycle analysis
dpo = detrended(close, 14)
current_dpo = dpo[-1]

if current_dpo > 0:
    # Price above trend cycle
else:
    # Price below trend cycle
```

#### Kaufman Adaptive Moving Average (KAMA)
```python
kama_values = kama(prices, period=10, fast_sc=2, slow_sc=30)
```
- **prices**: Price array
- **period**: Efficiency ratio period
- **fast_sc**: Fast smoothing constant
- **slow_sc**: Slow smoothing constant
- **Returns**: List of adaptive moving average values

```python
# Example: KAMA trend following
# Extract from internal buffer in callback function
def on_kline(kline):
    close_prices = [k["close"] for k in state["klines"]]
    
    if len(close_prices) >= 10:
        kama_line = kama(close_prices, 10, 2, 30)
        current_kama = kama_line[-1]
        current_price = kline.close
        
        if current_price > current_kama:
            # Uptrend confirmed
        else:
            # Downtrend or sideways
```

#### Chaikin Oscillator
```python
chaikin_values = chaikin_oscillator(high, low, close, volume, fast_period=3, slow_period=10)
```
- **high, low, close, volume**: Price and volume arrays
- **fast_period**: Fast EMA period
- **slow_period**: Slow EMA period
- **Returns**: List of Chaikin oscillator values

```python
# Example: Chaikin volume-price analysis
# Extract from internal buffer in callback function  
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    close_prices = [k["close"] for k in state["klines"]]
    volume_data = [k["volume"] for k in state["klines"]]
    
    if len(close_prices) >= 10:
        chaikin = chaikin_oscillator(high_prices, low_prices, close_prices, volume_data, 3, 10)
        current_chaikin = chaikin[-1]
        
        if current_chaikin > 0:
            # Accumulation (buying pressure)
        else:
            # Distribution (selling pressure)
```

#### Ultimate Oscillator
```python
ultimate_values = ultimate_oscillator(high, low, close, period1=7, period2=14, period3=28)
```
- **high, low, close**: Price arrays
- **period1, period2, period3**: Three timeframe periods
- **Returns**: List of Ultimate Oscillator values (0-100)

```python
# Example: Ultimate Oscillator multi-timeframe momentum
ultimate = ultimate_oscillator(high, low, close, 7, 14, 28)
current_ultimate = ultimate[-1]

if current_ultimate > 70:
    # Overbought across multiple timeframes
elif current_ultimate < 30:
    # Oversold across multiple timeframes
```

#### Heikin Ashi Candlesticks
```python
heikin_result = heikin_ashi(open, high, low, close)
```
- **open, high, low, close**: OHLC price arrays
- **Returns**: Dictionary with "open", "high", "low", "close" Heikin Ashi values

```python
# Example: Heikin Ashi trend analysis
ha = heikin_ashi(open, high, low, close)
ha_close = ha["close"][-1]
ha_open = ha["open"][-1]

if ha_close > ha_open:
    # Bullish Heikin Ashi candle
```

#### Vortex Indicator
```python
vortex_result = vortex(high, low, close, period=14)
```
- **high, low, close**: Price arrays
- **period**: Calculation period
- **Returns**: Dictionary with "vi_plus" and "vi_minus" arrays

```python
# Example: Vortex trend identification
vi = vortex(high, low, close, 14)
vi_plus = vi["vi_plus"][-1]
vi_minus = vi["vi_minus"][-1]

if vi_plus > vi_minus:
    # Bullish vortex signal
```

#### Williams Alligator
```python
alligator_result = williams_alligator(prices)
```
- **prices**: Price array (typically median price)
- **Returns**: Dictionary with "jaw", "teeth", "lips" lines

```python
# Example: Alligator trend system
alligator = williams_alligator(close)
jaw = alligator["jaw"][-1]
teeth = alligator["teeth"][-1] 
lips = alligator["lips"][-1]

# Trending when lines are apart
if lips > teeth > jaw:
    # Strong uptrend
elif lips < teeth < jaw:
    # Strong downtrend
```

#### Supertrend
```python
supertrend_result = supertrend(high, low, close, period=10, multiplier=3.0)
```
- **high, low, close**: Price arrays
- **period**: ATR period
- **multiplier**: ATR multiplier
- **Returns**: Dictionary with "supertrend" values and "trend" booleans

```python
# Example: Supertrend trading signals
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    close_prices = [k["close"] for k in state["klines"]]
    
    if len(close_prices) >= 20:
        st = supertrend(high_prices, low_prices, close_prices, 10, 3.0)
        supertrend_line = st["supertrend"][-1]
        is_uptrend = st["trend"][-1]
        current_price = kline.close
        
        if is_uptrend and current_price > supertrend_line:
            # Strong buy signal
        elif not is_uptrend and current_price < supertrend_line:
            # Strong sell signal
```

#### Stochastic RSI
```python
stoch_rsi_result = stochastic_rsi(prices, rsi_period=14, stoch_period=14, k_period=3, d_period=3)
```
- **prices**: Price array
- **rsi_period**: RSI calculation period
- **stoch_period**: Stochastic calculation period
- **k_period, d_period**: Smoothing periods
- **Returns**: Dictionary with "k" and "d" arrays

```python
# Example: Stochastic RSI momentum
stoch_rsi = stochastic_rsi(close, 14, 14, 3, 3)
k_line = stoch_rsi["k"][-1]
d_line = stoch_rsi["d"][-1]

if k_line > 80:
    # Overbought
elif k_line < 20:
    # Oversold
```

#### Awesome Oscillator
```python
ao_values = awesome_oscillator(high, low)
```
- **high, low**: Price arrays
- **Returns**: List of Awesome Oscillator values

```python
# Example: Awesome Oscillator momentum
ao = awesome_oscillator(high, low)
current_ao = ao[-1]
previous_ao = ao[-2]

if current_ao > 0 and previous_ao <= 0:
    # Zero line cross to positive (bullish)
```

#### Accelerator Oscillator
```python
ac_values = accelerator_oscillator(high, low, close)
```
- **high, low, close**: Price arrays
- **Returns**: List of Accelerator Oscillator values

```python
# Example: Accelerator Oscillator signals
ac = accelerator_oscillator(high, low, close)
current_ac = ac[-1]

if current_ac > 0:
    # Acceleration in current trend direction
```

#### Hull Moving Average (HMA)
```python
hma_values = hull_ma(prices, period)
```
- **prices**: List of price values
- **period**: HMA period (integer)
- **Returns**: List of Hull Moving Average values

```python
# Example: 21-period Hull Moving Average
hma21 = hull_ma(close, 21)
current_hma = hma21[-1]  # Latest HMA value
```

#### Weighted Moving Average (WMA)
```python
wma_values = wma(prices, period)
```
- **prices**: List of price values
- **period**: WMA period (integer)
- **Returns**: List of Weighted Moving Average values

```python
# Example: 10-period WMA
wma10 = wma(close, 10)
current_wma = wma10[-1]  # Latest WMA value
```

#### Arnaud Legoux Moving Average (ALMA)
```python
alma_values = alma(prices, period, offset=0.85, sigma=6.0)
```
- **prices**: List of price values
- **period**: ALMA period (integer)
- **offset**: Phase offset (default: 0.85)
- **sigma**: Smoothing factor (default: 6.0)
- **Returns**: List of ALMA values

```python
# Example: 21-period ALMA with default parameters
alma21 = alma(close, 21)
current_alma = alma21[-1]
```

#### Triple Exponential Moving Average (TEMA)
```python
tema_values = tema(prices, period)
```
- **prices**: List of price values
- **period**: TEMA period (integer)
- **Returns**: List of TEMA values with reduced lag

```python
# Example: 14-period TEMA
tema14 = tema(close, 14)
current_tema = tema14[-1]
```

### Momentum Indicators

#### Chande Momentum Oscillator (CMO)
```python
cmo_values = cmo(prices, period=14)
```
- **prices**: List of price values
- **period**: CMO period (default: 14)
- **Returns**: List of CMO values (-100 to 100)

```python
# Example: 14-period CMO
cmo14 = cmo(close, 14)
current_cmo = cmo14[-1]

if current_cmo > 50:
    # Strong bullish momentum
elif current_cmo < -50:
    # Strong bearish momentum
```

#### Know Sure Thing (KST)
```python
kst_result = kst(prices, roc1=10, roc2=15, roc3=20, roc4=30, sma1=10, sma2=10, sma3=10, sma4=15)
```
- **prices**: List of price values
- **roc1-4**: Rate of Change periods (default: 10,15,20,30)
- **sma1-4**: SMA smoothing periods (default: 10,10,10,15)
- **Returns**: Dictionary with "kst" and "signal" arrays

```python
# Example: KST oscillator with signal line
kst_data = kst(close)
kst_line = kst_data["kst"]
signal_line = kst_data["signal"]

# Check for bullish crossover
if kst_line[-1] > signal_line[-1] and kst_line[-2] <= signal_line[-2]:
    # KST crossed above signal line
```

#### Schaff Trend Cycle (STC)
```python
stc_values = stc(prices, fast_period=23, slow_period=50, cycle_period=10, factor=0.5)
```
- **prices**: List of price values
- **fast_period**: Fast MACD period (default: 23)
- **slow_period**: Slow MACD period (default: 50)
- **cycle_period**: Stochastic period (default: 10)
- **factor**: Smoothing factor (default: 0.5)
- **Returns**: List of STC values (0-100)

```python
# Example: STC for trend cycle detection
stc_line = stc(close, 12, 26, 10, 0.5)
current_stc = stc_line[-1]

if current_stc > 75:
    # Overbought trend cycle
elif current_stc < 25:
    # Oversold trend cycle
```

### Volatility Indicators

#### Chandelier Exit
```python
chandelier_result = chandelier_exit(high, low, close, period=22, multiplier=3.0)
```
- **high, low, close**: Price arrays
- **period**: ATR period (default: 22)
- **multiplier**: ATR multiplier (default: 3.0)
- **Returns**: Dictionary with "long_exit" and "short_exit" arrays

```python
# Example: Chandelier Exit levels
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    close_prices = [k["close"] for k in state["klines"]]
    
    if len(close_prices) >= 22:
        chandelier = chandelier_exit(high_prices, low_prices, close_prices, 22, 3.0)
        long_exit = chandelier["long_exit"][-1]
        short_exit = chandelier["short_exit"][-1]
        current_price = kline.close
        
        if current_price < long_exit:
            # Exit long position
        elif current_price > short_exit:
            # Exit short position
```

#### Chande Kroll Stop
```python
chande_result = chande_kroll_stop(high, low, close, period=10, multiplier=3.0)
```
- **high, low, close**: Price arrays  
- **period**: Period for calculation (default: 10)
- **multiplier**: ATR multiplier (default: 3.0)
- **Returns**: Dictionary with "long_stop" and "short_stop" arrays

```python
# Example: Chande Kroll Stop levels
ck_stop = chande_kroll_stop(high, low, close, 10, 3.0)
long_stop = ck_stop["long_stop"][-1]
short_stop = ck_stop["short_stop"][-1]
```

#### Price Channel
```python
channel_result = price_channel(high, low, period=20)
```
- **high, low**: Price arrays
- **period**: Channel period (default: 20)
- **Returns**: Dictionary with "upper", "middle", "lower" channel lines

```python
# Example: Price Channel breakout strategy
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    
    if len(high_prices) >= 20:
        channels = price_channel(high_prices, low_prices, 20)
        upper_channel = channels["upper"][-1]
        lower_channel = channels["lower"][-1]
        middle_channel = channels["middle"][-1]
        current_price = kline.close
        
        if current_price > upper_channel:
            # Upside breakout
        elif current_price < lower_channel:
            # Downside breakout
```

#### Mass Index
```python
mass_values = mass_index(high, low, period=9, sum_period=25)
```
- **high, low**: Price arrays
- **period**: EMA period for range calculation (default: 9)
- **sum_period**: Summation period (default: 25)
- **Returns**: List of Mass Index values

```python
# Example: Mass Index reversal detection
mass = mass_index(high, low, 9, 25)
current_mass = mass[-1]

if current_mass > 27:
    # Potential reversal signal
elif current_mass < 26.5:
    # Trend continuation likely
```

### Volume Indicators

#### Ease of Movement (EMV)
```python
emv_values = emv(high, low, close, volume, period=14)
```
- **high, low, close, volume**: Price and volume arrays
- **period**: Smoothing period (default: 14)
- **Returns**: List of EMV values

```python
# Example: Ease of Movement analysis
emv_line = emv(high, low, close, volume, 14)
current_emv = emv_line[-1]

if current_emv > 0:
    # Easy upward movement
else:
    # Difficult upward movement
```

#### Force Index
```python
fi_values = force_index(close, volume, period=13)
```
- **close, volume**: Price and volume arrays
- **period**: EMA smoothing period (default: 13)
- **Returns**: List of Force Index values

```python
# Example: Force Index trend confirmation
fi = force_index(close, volume, 13)
current_fi = fi[-1]

if current_fi > 0:
    # Buying pressure
else:
    # Selling pressure
```

#### Elder's Force Index
```python
elder_fi_result = elder_force_index(close, volume, short_period=2, long_period=13)
```
- **close, volume**: Price and volume arrays
- **short_period**: Short EMA period (default: 2)
- **long_period**: Long EMA period (default: 13)
- **Returns**: Dictionary with "short" and "long" Force Index arrays

```python
# Example: Elder's Force Index signals
elder_fi = elder_force_index(close, volume, 2, 13)
short_fi = elder_fi["short"][-1]
long_fi = elder_fi["long"][-1]

if short_fi > 0 and long_fi > 0:
    # Strong buying pressure
elif short_fi < 0 and long_fi < 0:
    # Strong selling pressure
```

#### Volume Oscillator
```python
vol_osc_values = volume_oscillator(volume, fast_period=5, slow_period=10)
```
- **volume**: Volume array
- **fast_period**: Fast MA period (default: 5)
- **slow_period**: Slow MA period (default: 10)
- **Returns**: List of volume oscillator values (percentage)

```python
# Example: Volume Oscillator trend detection
vol_osc = volume_oscillator(volume, 5, 10)
current_vol_osc = vol_osc[-1]

if current_vol_osc > 0:
    # Volume expanding
else:
    # Volume contracting
```

#### Volume Profile
```python
profile_data = volume_profile(high, low, close, volume, period=100, levels=20)
```
- **high, low, close, volume**: Price and volume arrays
- **period**: Number of bars to analyze (default: 100)
- **levels**: Number of price levels (default: 20)
- **Returns**: Dictionary mapping price levels to volume

```python
# Example: Volume Profile analysis
vp = volume_profile(high, low, close, volume, 100, 20)

# Find highest volume price level (Point of Control)
max_volume = 0
poc_price = 0
for price, vol in vp.items():
    if vol > max_volume:
        max_volume = vol
        poc_price = price

print(f"Point of Control: {poc_price}")
```

#### Klinger Volume Oscillator
```python
klinger_result = klinger_oscillator(high, low, close, volume, fast_period=34, slow_period=55, signal_period=13)
```
- **high, low, close, volume**: Price and volume arrays
- **fast_period**: Fast EMA period (default: 34)
- **slow_period**: Slow EMA period (default: 55)
- **signal_period**: Signal line period (default: 13)
- **Returns**: Dictionary with "oscillator" and "signal" arrays

```python
# Example: Klinger Oscillator signals
klinger = klinger_oscillator(high, low, close, volume, 34, 55, 13)
ko_line = klinger["oscillator"][-1]
signal_line = klinger["signal"][-1]

if ko_line > signal_line:
    # Accumulation signal
else:
    # Distribution signal
```

### Trend/Momentum Indicators

#### Balance of Power (BOP)
```python
bop_values = bop(open, high, low, close)
```
- **open, high, low, close**: OHLC price arrays
- **Returns**: List of BOP values (-1 to 1)

```python
# Example: Balance of Power analysis
bop_line = bop(open, high, low, close)
current_bop = bop_line[-1]

if current_bop > 0.5:
    # Strong buying pressure
elif current_bop < -0.5:
    # Strong selling pressure
```

#### Coppock Curve
```python
coppock_values = coppock_curve(prices, roc1_period=14, roc2_period=11, wma_period=10)
```
- **prices**: Price array
- **roc1_period**: First ROC period (default: 14)
- **roc2_period**: Second ROC period (default: 11)
- **wma_period**: WMA smoothing period (default: 10)
- **Returns**: List of Coppock Curve values

```python
# Example: Coppock Curve long-term reversal signals
coppock = coppock_curve(close, 14, 11, 10)
current_coppock = coppock[-1]
previous_coppock = coppock[-2]

if current_coppock > 0 and previous_coppock <= 0:
    # Bullish long-term reversal signal
```

## Additional Technical Indicators

The MarketMaestro strategy engine has been enhanced with 14 additional high-value technical indicators commonly used by professional traders:

### Momentum Indicators

#### Relative Vigor Index (RVI)
```python
rvi_result = rvi(open, high, low, close, period=14)
```
- **open, high, low, close**: OHLC price arrays
- **period**: RVI period (default: 14)
- **Returns**: Dictionary with "rvi" and "signal" arrays

```python
# Example: RVI momentum analysis
rvi_data = rvi(open, high, low, close, 14)
rvi_line = rvi_data["rvi"][-1]
signal_line = rvi_data["signal"][-1]

if rvi_line > signal_line:
    # Bullish momentum
```

#### Percentage Price Oscillator (PPO)
```python
ppo_result = ppo(prices, fast_period=12, slow_period=26, signal_period=9)
```
- **prices**: Price array
- **fast_period**: Fast EMA period (default: 12)
- **slow_period**: Slow EMA period (default: 26)
- **signal_period**: Signal line period (default: 9)
- **Returns**: Dictionary with "ppo", "signal", "histogram" arrays

```python
# Example: PPO analysis (percentage-based MACD)
ppo_data = ppo(close, 12, 26, 9)
ppo_line = ppo_data["ppo"][-1]
signal_line = ppo_data["signal"][-1]

if ppo_line > signal_line and ppo_line > 0:
    # Bullish momentum above zero line
```

### Volume-Based Indicators

#### Accumulation/Distribution Line (A/D)
```python
ad_values = accumulation_distribution(high, low, close, volume)
```
- **high, low, close, volume**: Price and volume arrays
- **Returns**: List of A/D line values

```python
# Example: A/D line trend confirmation
ad_line = accumulation_distribution(high, low, close, volume)
current_ad = ad_line[-1]
previous_ad = ad_line[-2]

if current_ad > previous_ad:
    # Accumulation (buying pressure)
```

#### Chaikin Money Flow (CMF)
```python
cmf_values = chaikin_money_flow(high, low, close, volume, period=20)
```
- **high, low, close, volume**: Price and volume arrays
- **period**: CMF period (default: 20)
- **Returns**: List of CMF values (-1 to 1)

```python
# Example: CMF buying/selling pressure
cmf_line = chaikin_money_flow(high, low, close, volume, 20)
current_cmf = cmf_line[-1]

if current_cmf > 0.1:
    # Strong buying pressure
elif current_cmf < -0.1:
    # Strong selling pressure
```

#### Money Flow Volume (MFV)
```python
mfv_values = money_flow_volume(high, low, close, volume)
```
- **high, low, close, volume**: Price and volume arrays
- **Returns**: List of money flow volume values

```python
# Example: Money flow volume analysis
mfv = money_flow_volume(high, low, close, volume)
current_mfv = mfv[-1]

if current_mfv > 0:
    # Positive money flow (buying pressure)
```

#### Williams Accumulation/Distribution (Williams A/D)
```python
wad_values = williams_ad(high, low, close)
```
- **high, low, close**: Price arrays
- **Returns**: List of Williams A/D values

```python
# Example: Williams A/D analysis
wad = williams_ad(high, low, close)
trend = wad[-1] - wad[-10]  # 10-period trend

if trend > 0:
    # Accumulation trend
```

### Statistical & Regression Indicators

#### Linear Regression
```python
lr_values = linear_regression(prices, period)
```
- **prices**: Price array
- **period**: Regression period
- **Returns**: List of linear regression values

```python
# Example: Linear regression trend line
lr = linear_regression(close, 14)
current_lr = lr[-1]
current_price = close[-1]

if current_price > current_lr:
    # Price above regression line (bullish)
```

#### Linear Regression Slope
```python
slope_values = linear_regression_slope(prices, period)
```
- **prices**: Price array
- **period**: Regression period
- **Returns**: List of slope values

```python
# Example: Trend direction via slope
slope = linear_regression_slope(close, 14)
current_slope = slope[-1]

if current_slope > 0:
    # Upward trend
elif current_slope < 0:
    # Downward trend
```

#### Correlation Coefficient
```python
corr_values = correlation_coefficient(prices, period)
```
- **prices**: Price array
- **period**: Correlation period
- **Returns**: List of correlation values (-1 to 1)

```python
# Example: Price-time correlation
corr = correlation_coefficient(close, 14)
current_corr = corr[-1]

if current_corr > 0.8:
    # Strong positive trend
elif current_corr < -0.8:
    # Strong negative trend
```

#### Standard Error
```python
se_values = standard_error(prices, period)
```
- **prices**: Price array
- **period**: Regression period
- **Returns**: List of standard error values

```python
# Example: Trend reliability via standard error
se = standard_error(close, 14)
current_se = se[-1]

# Lower standard error = more reliable trend
if current_se < threshold:
    # High confidence in trend
```

### Bollinger Band Derivatives

#### Bollinger %B
```python
percent_b_values = bollinger_percent_b(prices, period=20, multiplier=2.0)
```
- **prices**: Price array
- **period**: BB period (default: 20)
- **multiplier**: Standard deviation multiplier (default: 2.0)
- **Returns**: List of %B values

```python
# Example: Bollinger %B position analysis
percent_b = bollinger_percent_b(close, 20, 2.0)
current_b = percent_b[-1]

if current_b > 1.0:
    # Price above upper band
elif current_b < 0.0:
    # Price below lower band
elif current_b > 0.8:
    # Near upper band (overbought)
elif current_b < 0.2:
    # Near lower band (oversold)
```

#### Bollinger Band Width
```python
bbw_values = bollinger_band_width(prices, period=20, multiplier=2.0)
```
- **prices**: Price array
- **period**: BB period (default: 20)
- **multiplier**: Standard deviation multiplier (default: 2.0)
- **Returns**: List of band width values

```python
# Example: Volatility analysis via band width
bbw = bollinger_band_width(close, 20, 2.0)
current_width = bbw[-1]
avg_width = sum(bbw[-20:]) / 20

if current_width < avg_width * 0.5:
    # Band squeeze (low volatility)
elif current_width > avg_width * 1.5:
    # Band expansion (high volatility)
```

### Volatility Indicators

#### Price Rate of Change (Price ROC)
```python
price_roc_values = price_roc(prices, period)
```
- **prices**: Price array
- **period**: ROC period
- **Returns**: List of price ROC values (percentage)

```python
# Example: Price momentum analysis
proc = price_roc(close, 10)
current_roc = proc[-1]

if current_roc > 5:
    # Strong upward momentum (>5%)
elif current_roc < -5:
    # Strong downward momentum (<-5%)
```

#### Volatility Index
```python
vi_values = volatility_index(high, low, close, period=30)
```
- **high, low, close**: Price arrays
- **period**: Volatility period (default: 30)
- **Returns**: List of volatility index values (annualized %)

```python
# Example: Market volatility assessment
vi = volatility_index(high, low, close, 30)
current_vol = vi[-1]

if current_vol > 50:
    # High volatility environment
elif current_vol < 20:
    # Low volatility environment
```

### Advanced Usage Examples

#### Multi-Indicator Confirmation Strategy
```python
def on_kline(kline):
    init_state()
    cfg = get_config_values()
    
    # Get state
    klines = get_state("klines", [])
    # ... update klines ...
    
    # Extract price arrays
    close_prices = [k["close"] for k in klines]
    high_prices = [k["high"] for k in klines]
    low_prices = [k["low"] for k in klines]
    volume_data = [k["volume"] for k in klines]
    
    if len(close_prices) < 30:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Multiple indicator confirmation
    rvi_data = rvi(open_prices, high_prices, low_prices, close_prices, 14)
    cmf_line = chaikin_money_flow(high_prices, low_prices, close_prices, volume_data, 20)
    lr_slope = linear_regression_slope(close_prices, 14)
    
    rvi_bullish = rvi_data["rvi"][-1] > rvi_data["signal"][-1]
    cmf_bullish = cmf_line[-1] > 0.1
    trend_bullish = lr_slope[-1] > 0
    
    confirmations = sum([rvi_bullish, cmf_bullish, trend_bullish])
    
    if confirmations >= 2:
        return {
            "action": "buy",
            "quantity": cfg["position_size"],
            "reason": f"Multi-indicator confirmation: {confirmations}/3"
        }
    
    return {"action": "hold"}
```

#### Bollinger Band Squeeze Strategy
```python
def on_kline(kline):
    init_state()
    cfg = get_config_values()
    
    # Get state and update
    klines = get_state("klines", [])
    # ... update klines ...
    
    close_prices = [k["close"] for k in klines]
    
    if len(close_prices) < 30:
        return {"action": "hold"}
    
    # Bollinger Band analysis
    bbw = bollinger_band_width(close_prices, 20, 2.0)
    percent_b = bollinger_percent_b(close_prices, 20, 2.0)
    
    # Calculate squeeze conditions
    current_width = bbw[-1]
    avg_width = sum(bbw[-20:]) / 20
    is_squeeze = current_width < avg_width * 0.7
    
    current_b = percent_b[-1]
    
    # Strategy logic
    if is_squeeze and current_b > 0.8:
        # Squeeze with price near upper band
        return {
            "action": "buy",
            "quantity": cfg["position_size"],
            "reason": f"Squeeze breakout: width={current_width:.4f}, %B={current_b:.2f}"
        }
    elif is_squeeze and current_b < 0.2:
        # Squeeze with price near lower band
        return {
            "action": "sell",
            "quantity": cfg["position_size"],
            "reason": f"Squeeze breakdown: width={current_width:.4f}, %B={current_b:.2f}"
        }
    
    return {"action": "hold"}
```

### All Available Indicators Summary
- **Basic**: `sma`, `ema`, `rsi`, `stddev`, `roc`
- **Moving Averages**: `sma`, `ema`, `wma`, `hull_ma`, `alma`, `tema`
- **Advanced Momentum**: `macd`, `stochastic`, `williams_r`, `cci`, `mfi`, `tsi`, `ultimate_oscillator`, `stochastic_rsi`, `cmo`, `kst`, `stc`, `rvi`, `ppo`
- **Volatility**: `bollinger`, `atr`, `keltner`, `donchian`, `supertrend`, `chandelier_exit`, `chande_kroll_stop`, `price_channel`, `mass_index`, `bollinger_percent_b`, `bollinger_band_width`, `standard_error`, `volatility_index`
- **Volume**: `vwap`, `obv`, `mfi`, `chaikin_oscillator`, `emv`, `force_index`, `elder_force_index`, `volume_oscillator`, `volume_profile`, `klinger_oscillator`, `chaikin_money_flow`, `accumulation_distribution`, `money_flow_volume`, `williams_ad`
- **Trend**: `adx`, `parabolic_sar`, `ichimoku`, `aroon`, `kama`, `detrended`, `williams_alligator`, `vortex`, `bop`, `coppock_curve`, `linear_regression`, `linear_regression_slope`, `correlation_coefficient`
- **Support/Resistance**: `pivot_points`, `fibonacci`
- **Advanced Analysis**: `advanced_cci`, `elder_ray`
- **Candlestick Patterns**: `heikin_ashi`
- **Oscillators**: `awesome_oscillator`, `accelerator_oscillator`
- **Statistical**: `price_roc`

## Utility Functions

### Price Analysis
```python
# Rolling highest/lowest values
highest_values = highest(prices, period)
lowest_values = lowest(prices, period)

# Example: Find highest high in last 20 periods
# Extract from internal buffer in callback function
def on_kline(kline):
    high_prices = [k["high"] for k in state["klines"]]
    
    if len(high_prices) >= 20:
        recent_high_values = highest(high_prices, 20)
        recent_high = recent_high_values[-1]
```

### Signal Detection
```python
# Detect when series1 crosses above series2
crossover_signals = crossover(series1, series2)

# Detect when series1 crosses below series2  
crossunder_signals = crossunder(series1, series2)

# Example: MA crossover detection
# Extract from internal buffer in callback function
def on_kline(kline):
    close_prices = [k["close"] for k in state["klines"]]
    
    if len(close_prices) >= 20:
        short_ma = sma(close_prices, 10)
        long_ma = sma(close_prices, 20)
        bullish_cross = crossover(short_ma, long_ma)
        
        if len(bullish_cross) > 0 and bullish_cross[-1]:  # Latest value is True
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
    "indicators": {},
    "klines": []  # Internal buffer for price data
}

def on_kline(kline):
    # Update internal buffer
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep buffer size manageable
    if len(state["klines"]) > 100:
        state["klines"] = state["klines"][-100:]
    
    # Extract price arrays for calculations
    close_prices = [k["close"] for k in state["klines"]]
    
    # Cache expensive calculations
    if len(close_prices) >= 20:
        state["indicators"]["sma20"] = sma(close_prices, 20)
        state["indicators"]["rsi"] = rsi(close_prices, 14)
```

### 2. Data Validation
```python
def on_kline(kline):
    # Maintain internal buffer
    state["klines"].append({
        "close": kline.close,
        "high": kline.high,
        "low": kline.low,
        # ... other data
    })
    
    # Extract prices from internal buffer  
    close_prices = [k["close"] for k in state["klines"]]
    
    # Always validate data availability
    if len(close_prices) < 20:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Check for valid indicator values
    current_rsi = rsi(close_prices, 14)[-1]
    if math.isnan(current_rsi):
        return {"action": "hold", "reason": "Invalid RSI"}
```

### 3. Risk Management
```python
def calculate_position_size(kline, risk_percent=1.0):
    """Calculate position size based on risk percentage"""
    account_balance = 1000.0  # Get from config or API
    risk_amount = account_balance * (risk_percent / 100)
    
    # Extract price data from internal buffer
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    close_prices = [k["close"] for k in state["klines"]]
    
    # Calculate stop loss distance using ATR
    if len(close_prices) >= 14:
        atr_values = atr(high_prices, low_prices, close_prices, 14)
        atr_value = atr_values[-1]
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
    
    # Extract price data from internal buffer
    close_prices = [k["close"] for k in state["klines"]]
    volume_data = [k["volume"] for k in state["klines"]]
    
    if len(close_prices) < 20:
        return False
    
    # RSI confirmation
    rsi_values = rsi(close_prices, 14)
    current_rsi = rsi_values[-1]
    if primary_signal == "buy" and current_rsi < 50:
        confirmations += 1
    elif primary_signal == "sell" and current_rsi > 50:
        confirmations += 1
    
    # Volume confirmation
    if len(volume_data) >= 20:
        current_volume = volume_data[-1]
        avg_volume = sma(volume_data, 20)[-1]
        if current_volume > avg_volume * 1.5:
            confirmations += 1
    
    return confirmations >= 2  # Require at least 2 confirmations
```

## Complete Examples

### Example 1: Thread-Safe RSI Strategy
```python
# RSI Strategy using new thread-safe architecture

def settings():
    return {
        "interval": "5m",        # Default interval - can be overridden in config
        "rsi_period": 14,
        "oversold": 30,
        "overbought": 70,
        "position_size": 0.01
    }

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    params = settings()
    return {
        "rsi_period": get_config("rsi_period", params["rsi_period"]),
        "oversold": get_config("oversold", params["oversold"]),
        "overbought": get_config("overbought", params["overbought"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("position", 0)
        set_state("entry_price", 0.0)
        set_state("rsi_values", [])
        set_state("last_signal", "hold")

def on_kline(kline):
    """Handle new kline data"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    position = get_state("position", 0)
    entry_price = get_state("entry_price", 0.0)
    
    # Add new kline to buffer
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only what we need for calculations
    max_needed = cfg["rsi_period"] + 10
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Check data availability
    if len(klines) < cfg["rsi_period"] + 1:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract close prices
    close_prices = [k["close"] for k in klines]
    
    # Calculate RSI
    rsi_values = rsi(close_prices, cfg["rsi_period"])
    current_rsi = rsi_values[-1]
    current_price = kline.close
    
    # Store RSI values in state
    set_state("rsi_values", rsi_values)
    
    # Entry signals
    if position == 0:  # No position
        if current_rsi < cfg["oversold"]:
            set_state("position", 1)
            set_state("entry_price", current_price)
            set_state("last_signal", "buy")
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": f"RSI oversold: {round(current_rsi, 2)}"
            }
    
    # Exit signals
    elif position == 1:  # Long position
        if current_rsi > cfg["overbought"]:
            set_state("position", 0)
            set_state("entry_price", 0.0)
            set_state("last_signal", "sell")
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": f"RSI overbought: {round(current_rsi, 2)}"
            }
    
    return {"action": "hold", "reason": f"RSI: {round(current_rsi, 2)}"}

def on_orderbook(orderbook):
    """Handle orderbook updates for spread analysis"""
    if len(orderbook.bids) > 0 and len(orderbook.asks) > 0:
        spread = orderbook.asks[0].price - orderbook.bids[0].price
        mid_price = (orderbook.bids[0].price + orderbook.asks[0].price) / 2
        spread_percent = (spread / mid_price) * 100
        
        # Only trade when spread is reasonable
        if spread_percent > 0.1:
            return {
                "action": "hold",
                "reason": f"Spread too wide: {round(spread_percent, 3)}%"
            }
    
    return {"action": "hold", "reason": "Orderbook conditions acceptable"}

def on_ticker(ticker):
    """Handle ticker updates for volume confirmation"""
    return {"action": "hold", "reason": "No ticker signals"}
```

### Example 2: Thread-Safe SMA Crossover Strategy
```python
# Simple Moving Average Crossover Strategy

def settings():
    return {
        "interval": "15m",       # Default interval
        "short_period": 10,
        "long_period": 20,
        "position_size": 0.01
    }

def get_config_values():
    """Get configuration values with fallbacks"""
    params = settings()
    return {
        "short_period": get_config("short_period", params["short_period"]),
        "long_period": get_config("long_period", params["long_period"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize thread-safe state"""
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("short_ma", [])
        set_state("long_ma", [])
        set_state("last_signal", "hold")
        set_state("position", 0)

def on_kline(kline):
    """Handle new kline data"""
    init_state()
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    
    # Add new kline
    klines.append({
        "timestamp": kline.timestamp,
        "close": kline.close,
        "high": kline.high,
        "low": kline.low,
        "volume": kline.volume
    })
    
    # Keep required history
    max_needed = cfg["long_period"] + 10
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    set_state("klines", klines)
    
    # Extract close prices
    close_prices = [k["close"] for k in klines]
    
    # Calculate moving averages
    if len(close_prices) >= cfg["short_period"]:
        short_ma = sma(close_prices, cfg["short_period"])
        set_state("short_ma", short_ma)
    
    if len(close_prices) >= cfg["long_period"]:
        long_ma = sma(close_prices, cfg["long_period"])
        set_state("long_ma", long_ma)
    
    # Get MA values from state
    short_ma = get_state("short_ma", [])
    long_ma = get_state("long_ma", [])
    last_signal = get_state("last_signal", "hold")
    
    # Check for crossover signals
    if len(short_ma) >= 2 and len(long_ma) >= 2:
        current_short = short_ma[-1]
        current_long = long_ma[-1]
        prev_short = short_ma[-2]
        prev_long = long_ma[-2]
        
        # Bullish crossover: short MA crosses above long MA
        if prev_short <= prev_long and current_short > current_long and last_signal != "buy":
            set_state("last_signal", "buy")
            set_state("position", 1)
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": f"MA bullish crossover: {round(current_short, 2)} > {round(current_long, 2)}"
            }
        
        # Bearish crossover: short MA crosses below long MA  
        elif prev_short >= prev_long and current_short < current_long and last_signal != "sell":
            set_state("last_signal", "sell")
            set_state("position", 0)
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": f"MA bearish crossover: {round(current_short, 2)} < {round(current_long, 2)}"
            }
    
    return {"action": "hold", "reason": "No crossover signal"}
```
    
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
    "klines": [],
    "trend": "none",
    "position": 0
}

def on_kline(kline):
    # Maintain internal kline buffer
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Configuration
    fast_period = config.get("fast_ma", 12)
    slow_period = config.get("slow_ma", 26)
    
    # Data validation - keep sufficient buffer
    min_periods = max(fast_period, slow_period, 14) + 5
    if len(state["klines"]) > min_periods * 2:
        state["klines"] = state["klines"][-min_periods * 2:]
    
    if len(state["klines"]) < min_periods:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract price arrays from internal buffer
    close_prices = [k["close"] for k in state["klines"]]
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    
    # Calculate indicators
    fast_ma = ema(close_prices, fast_period)
    slow_ma = ema(close_prices, slow_period)
    rsi_values = rsi(close_prices, 14)
    atr_values = atr(high_prices, low_prices, close_prices, 14)
    
    current_price = kline.close
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
    
```

### Example 3: Advanced Multi-Timeframe Strategy
```python
# Advanced strategy with multi-timeframe analysis and risk management

def settings():
    return {
        "interval": "5m",        # Trading timeframe
        "htf_interval": "1h",    # Higher timeframe for trend
        "rsi_period": 14,
        "ema_fast": 12,
        "ema_slow": 26,
        "position_size": 0.01,
        "risk_reward": 2.0,
        "max_trades_per_day": 3
    }

def get_config_values():
    """Get configuration with thread-safe access"""
    params = settings()
    return {
        "rsi_period": get_config("rsi_period", params["rsi_period"]),
        "ema_fast": get_config("ema_fast", params["ema_fast"]),
        "ema_slow": get_config("ema_slow", params["ema_slow"]),
        "position_size": get_config("position_size", params["position_size"]),
        "risk_reward": get_config("risk_reward", params["risk_reward"]),
        "max_trades_per_day": get_config("max_trades_per_day", params["max_trades_per_day"])
    }

def init_state():
    """Initialize comprehensive state management"""
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("position", 0)
        set_state("entry_price", 0.0)
        set_state("stop_loss", 0.0)
        set_state("take_profit", 0.0)
        set_state("trades_today", 0)
        set_state("last_trade_date", "")

def check_daily_reset():
    """Reset daily counters"""
    from datetime import datetime
    today = datetime.now().strftime("%Y-%m-%d")
    last_date = get_state("last_trade_date", "")
    
    if today != last_date:
        set_state("trades_today", 0)
        set_state("last_trade_date", today)

def calculate_position_size(current_price, stop_loss_price):
    """Dynamic position sizing based on risk"""
    cfg = get_config_values()
    risk_per_trade = 0.01  # 1% risk per trade
    
    if stop_loss_price <= 0:
        return cfg["position_size"]
    
    risk_amount = current_price * risk_per_trade
    price_diff = abs(current_price - stop_loss_price)
    
    if price_diff > 0:
        position_size = risk_amount / price_diff
        return min(position_size, cfg["position_size"] * 2)  # Cap at 2x normal size
    
    return cfg["position_size"]

def on_kline(kline):
    """Main strategy logic with risk management"""
    init_state()
    check_daily_reset()
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    position = get_state("position", 0)
    trades_today = get_state("trades_today", 0)
    
    # Check daily trade limit
    if trades_today >= cfg["max_trades_per_day"]:
        return {"action": "hold", "reason": "Daily trade limit reached"}
    
    # Add new kline
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Maintain buffer size
    max_needed = max(cfg["rsi_period"], cfg["ema_slow"]) + 20
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    set_state("klines", klines)
    
    # Check data sufficiency
    if len(klines) < cfg["rsi_period"] + 1:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract prices
    close_prices = [k["close"] for k in klines]
    high_prices = [k["high"] for k in klines]
    low_prices = [k["low"] for k in klines]
    
    # Calculate indicators
    rsi_values = rsi(close_prices, cfg["rsi_period"])
    ema_fast_values = ema(close_prices, cfg["ema_fast"])
    ema_slow_values = ema(close_prices, cfg["ema_slow"])
    
    current_rsi = rsi_values[-1]
    current_ema_fast = ema_fast_values[-1]
    current_ema_slow = ema_slow_values[-1]
    current_price = kline.close
    
    # Entry logic
    if position == 0:
        # Long setup: EMA bullish + RSI oversold
        if (current_ema_fast > current_ema_slow and 
            current_rsi < 40):
            
            # Calculate stop loss and take profit
            recent_low = min(low_prices[-10:])
            stop_loss = recent_low * 0.99  # 1% below recent low
            risk_amount = current_price - stop_loss
            take_profit = current_price + (risk_amount * cfg["risk_reward"])
            
            # Calculate position size
            position_size = calculate_position_size(current_price, stop_loss)
            
            # Update state
            set_state("position", 1)
            set_state("entry_price", current_price)
            set_state("stop_loss", stop_loss)
            set_state("take_profit", take_profit)
            set_state("trades_today", trades_today + 1)
            
            return {
                "action": "buy",
                "quantity": position_size,
                "type": "market",
                "stop_loss": stop_loss,
                "take_profit": take_profit,
                "reason": f"Long setup: RSI={round(current_rsi, 2)}"
            }
    
    # Exit logic for active positions
    elif position != 0:
        stop_loss = get_state("stop_loss", 0.0)
        take_profit = get_state("take_profit", 0.0)
        
        # Check stop loss and take profit
        if position == 1:  # Long position
            if current_price <= stop_loss:
                set_state("position", 0)
                return {
                    "action": "sell",
                    "quantity": cfg["position_size"],
                    "type": "market",
                    "reason": "Stop loss hit"
                }
            elif current_price >= take_profit:
                set_state("position", 0)
                return {
                    "action": "sell",
                    "quantity": cfg["position_size"],
                    "type": "market",
                    "reason": "Take profit hit"
                }
    
    return {
        "action": "hold", 
        "reason": f"Monitoring: RSI={round(current_rsi, 2)}"
    }
```

## Performance Monitoring

The strategy engine provides comprehensive performance tracking and risk management:

### State Management
- Thread-safe state operations with `get_state()` and `set_state()`
- Automatic state persistence across restarts
- Per-strategy, per-symbol state isolation

### Performance Metrics
- Real-time P&L tracking
- Win/loss ratios
- Maximum drawdown monitoring
- Sharpe ratio calculations

### Risk Controls
- Position size limits
- Daily trade limits
- Maximum concurrent positions
- Stop-loss enforcement

### Configuration Override Examples

```yaml
# config.yaml - Global defaults with per-symbol overrides
strategies:
  rsi_strategy:
    default_config:
      rsi_period: 14
      oversold: 30
      overbought: 70
      position_size: 0.01
    
    # Per-symbol overrides
    symbol_overrides:
      BTCUSDT:
        rsi_period: 21      # Different RSI period for BTC
        position_size: 0.005 # Smaller position for BTC
      
      ETHUSDT:
        oversold: 25        # More aggressive oversold level
        overbought: 75      # More aggressive overbought level
```

## Error Handling

### Common Issues and Solutions

#### 1. Insufficient Data
```python
# Always check data length before indicator calculations
def on_kline(kline):
    klines = get_state("klines", [])
    close_prices = [k["close"] for k in klines]
    
    required_periods = 20  # Or whatever your strategy needs
    if len(close_prices) < required_periods:
        return {"action": "hold", "reason": "Insufficient data"}
```

#### 2. NaN Values
```python
import math

indicator_value = rsi(close, 14)[-1]
if math.isnan(indicator_value):
    return {"action": "hold", "reason": "Invalid indicator value"}
```

#### 3. Thread-Safe State Access
```python
# Always use get_state/set_state for thread safety
def on_kline(kline):
    # Correct: Thread-safe state access
    position = get_state("position", 0)
    set_state("position", 1)
    
    # Incorrect: Direct state manipulation (not thread-safe)
    # state["position"] = 1  # DON'T DO THIS
```

### Debug Logging
```python
def on_kline(kline):
    # Use log() for structured debugging
    log(f"Processing kline: {kline.timestamp}, price: {kline.close}")
    
    # Use print() for simple debugging
    print(f"Current position: {get_state('position', 0)}")
```

---

This documentation covers the complete MarketMaestro strategy engine with thread-safe architecture. All examples use the new `get_state()`, `set_state()`, and `get_config()` functions for safe concurrent operation.

For additional examples, see the strategy files in the `/strategy` directory.

For questions or issues, refer to the main [README](../README.md) or check the source code documentation.