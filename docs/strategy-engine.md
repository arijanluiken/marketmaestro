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

### on_start()
**Optional**: For strategy initialization  
**Purpose**: Initialize strategy state, validate configuration, setup variables  
**Frequency**: Called once when strategy starts

```python
def on_start():
    """Initialize strategy when it starts"""
    print("🚀 Strategy starting")
    print("📊 Initializing technical analysis parameters")
    
    # Validate configuration
    if config.get("risk_percent", 0) > 5:
        print("⚠️  High risk percentage detected")
    
    # Initialize global state (if needed)
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
channels = donchian(high, low, 20)
upper_channel = channels["upper"][-1]
lower_channel = channels["lower"][-1]
current_price = close[-1]

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
kama_line = kama(close, 10, 2, 30)
current_kama = kama_line[-1]
current_price = close[-1]

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
chaikin = chaikin_oscillator(high, low, close, volume, 3, 10)
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
st = supertrend(high, low, close, 10, 3.0)
supertrend_line = st["supertrend"][-1]
is_uptrend = st["trend"][-1]
current_price = close[-1]

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
chandelier = chandelier_exit(high, low, close, 22, 3.0)
long_exit = chandelier["long_exit"][-1]
short_exit = chandelier["short_exit"][-1]
current_price = close[-1]

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
channels = price_channel(high, low, 20)
upper_channel = channels["upper"][-1]
lower_channel = channels["lower"][-1]
middle_channel = channels["middle"][-1]
current_price = close[-1]

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

### All Available Indicators Summary
- **Basic**: `sma`, `ema`, `rsi`, `stddev`, `roc`
- **Moving Averages**: `sma`, `ema`, `wma`, `hull_ma`, `alma`, `tema`
- **Advanced Momentum**: `macd`, `stochastic`, `williams_r`, `cci`, `mfi`, `tsi`, `ultimate_oscillator`, `stochastic_rsi`, `cmo`, `kst`, `stc`
- **Volatility**: `bollinger`, `atr`, `keltner`, `donchian`, `supertrend`, `chandelier_exit`, `chande_kroll_stop`, `price_channel`, `mass_index`
- **Volume**: `vwap`, `obv`, `mfi`, `chaikin_oscillator`, `emv`, `force_index`, `elder_force_index`, `volume_oscillator`, `volume_profile`, `klinger_oscillator`
- **Trend**: `adx`, `parabolic_sar`, `ichimoku`, `aroon`, `kama`, `detrended`, `williams_alligator`, `vortex`, `bop`, `coppock_curve`
- **Support/Resistance**: `pivot_points`, `fibonacci`
- **Advanced Analysis**: `advanced_cci`, `elder_ray`
- **Candlestick Patterns**: `heikin_ashi`
- **Oscillators**: `awesome_oscillator`, `accelerator_oscillator`

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