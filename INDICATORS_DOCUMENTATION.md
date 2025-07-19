# Enhanced Technical Indicators Documentation

The trading strategy engine has been enhanced with several new technical indicators. Here's a comprehensive guide to all available indicators:

## New Advanced Indicators

### 1. On-Balance Volume (OBV)
**Function:** `obv(close, volume)`
**Purpose:** Volume-based momentum indicator
**Usage:**
```python
obv_line = obv(close, volume)
```
**Description:** Tracks cumulative volume flow based on price movements. Rising OBV suggests buying pressure, falling OBV suggests selling pressure.

### 2. Average Directional Index (ADX)
**Function:** `adx(high, low, close, period)`
**Purpose:** Measures trend strength and directional movement
**Usage:**
```python
adx_result = adx(high, low, close, 14)
adx_line = adx_result["adx"]
plus_di = adx_result["plus_di"]
minus_di = adx_result["minus_di"]
```
**Description:** 
- ADX > 25: Strong trend
- ADX < 20: Weak trend/sideways market
- +DI > -DI: Bullish momentum
- -DI > +DI: Bearish momentum

### 3. Parabolic Stop and Reverse (SAR)
**Function:** `parabolic_sar(high, low, step=0.02, max_step=0.2)`
**Purpose:** Trend-following indicator for stop-loss levels
**Usage:**
```python
psar = parabolic_sar(high, low, step=0.02, max_step=0.2)
```
**Description:** 
- Price above PSAR: Uptrend
- Price below PSAR: Downtrend
- PSAR reversal indicates potential trend change

### 4. Keltner Channels
**Function:** `keltner(high, low, close, period, multiplier=2.0)`
**Purpose:** Volatility-based channel indicator
**Usage:**
```python
keltner_result = keltner(high, low, close, 20, multiplier=2.0)
keltner_upper = keltner_result["upper"]
keltner_middle = keltner_result["middle"]
keltner_lower = keltner_result["lower"]
```
**Description:** 
- Price above upper band: Potential overbought
- Price below lower band: Potential oversold
- Middle line is EMA-based trend line

### 5. Ichimoku Cloud
**Function:** `ichimoku(high, low, close, conversion_period=9, base_period=26, span_b_period=52, displacement=26)`
**Purpose:** Comprehensive trend and momentum system
**Usage:**
```python
ichimoku_result = ichimoku(high, low, close)
tenkan_sen = ichimoku_result["tenkan_sen"]      # Conversion Line
kijun_sen = ichimoku_result["kijun_sen"]        # Base Line
senkou_span_a = ichimoku_result["senkou_span_a"] # Leading Span A
senkou_span_b = ichimoku_result["senkou_span_b"] # Leading Span B
chikou_span = ichimoku_result["chikou_span"]     # Lagging Span
```
**Description:** 
- Tenkan > Kijun: Bullish signal
- Price above cloud: Uptrend
- Price below cloud: Downtrend
- Cloud thickness indicates support/resistance strength

### 6. Pivot Points
**Function:** `pivot_points(high, low, close)`
**Purpose:** Support and resistance levels based on previous period
**Usage:**
```python
pivot_result = pivot_points(high, low, close)
pivot = pivot_result["pivot"]
r1 = pivot_result["r1"]     # Resistance 1
r2 = pivot_result["r2"]     # Resistance 2
r3 = pivot_result["r3"]     # Resistance 3
s1 = pivot_result["s1"]     # Support 1
s2 = pivot_result["s2"]     # Support 2
s3 = pivot_result["s3"]     # Support 3
```
**Description:** Key support and resistance levels calculated from previous high, low, and close.

### 7. Fibonacci Retracement
**Function:** `fibonacci(high, low)`
**Purpose:** Calculate Fibonacci retracement levels
**Usage:**
```python
fib_levels = fibonacci(recent_high, recent_low)
fib_236 = fib_levels["23.6"]
fib_382 = fib_levels["38.2"]
fib_500 = fib_levels["50.0"]
fib_618 = fib_levels["61.8"]
fib_786 = fib_levels["78.6"]
```
**Description:** Key retracement levels for identifying potential reversal zones.

### 8. Aroon Oscillator
**Function:** `aroon(high, low, period)`
**Purpose:** Momentum indicator measuring trend strength
**Usage:**
```python
aroon_result = aroon(high, low, 14)
aroon_up = aroon_result["aroon_up"]
aroon_down = aroon_result["aroon_down"]
```
**Description:** 
- Aroon Up > 70 & Aroon Down < 30: Strong uptrend
- Aroon Down > 70 & Aroon Up < 30: Strong downtrend
- Both near 50: Consolidation

## Existing Indicators (Enhanced)

### Basic Indicators
- `sma(prices, period)` - Simple Moving Average
- `ema(prices, period)` - Exponential Moving Average
- `rsi(prices, period)` - Relative Strength Index
- `stddev(prices, period)` - Standard Deviation
- `roc(prices, period)` - Rate of Change

### Advanced Existing Indicators
- `macd(prices, fast_period, slow_period, signal_period)` - MACD
- `bollinger(prices, period, multiplier)` - Bollinger Bands
- `stochastic(high, low, close, k_period, d_period)` - Stochastic Oscillator
- `williams_r(high, low, close, period)` - Williams %R
- `atr(high, low, close, period)` - Average True Range
- `cci(high, low, close, period)` - Commodity Channel Index
- `vwap(high, low, close, volume)` - Volume Weighted Average Price
- `mfi(high, low, close, volume, period)` - Money Flow Index

### Utility Functions
- `highest(prices, period)` - Rolling highest values
- `lowest(prices, period)` - Rolling lowest values
- `crossover(series1, series2)` - Detect crossovers
- `crossunder(series1, series2)` - Detect crossunders

## Strategy Examples

### 1. Advanced Multi-Indicator Strategy
See: `strategy/advanced_indicators_demo.star`
- Uses ADX, Parabolic SAR, Keltner Channels, OBV, Aroon, and Ichimoku
- Combines multiple signals for robust trading decisions

### 2. Support/Resistance Strategy  
See: `strategy/support_resistance.star`
- Uses Pivot Points and Fibonacci levels
- Focuses on key support and resistance zones

### 3. Indicator Testing
See: `strategy/indicator_test.star`
- Tests all new indicators with sample data
- Useful for verifying indicator functionality

## Best Practices

1. **Combine Multiple Indicators:** Use 2-3 complementary indicators to avoid false signals
2. **Consider Market Context:** Trend-following indicators work better in trending markets
3. **Use Appropriate Timeframes:** Higher timeframes provide more reliable signals
4. **Risk Management:** Always implement proper stop-losses and position sizing
5. **Backtesting:** Test strategies thoroughly before live trading

## Performance Considerations

- Indicators with longer periods require more historical data
- Complex indicators (like Ichimoku) may have higher computational overhead
- Cache indicator results when possible to avoid recalculation

## Error Handling

All indicators return `NaN` values for periods where insufficient data is available. Always check for valid values before using in trading logic:

```python
if current_adx and not math.isnan(current_adx):
    # Use the indicator value
    pass
```

This comprehensive set of indicators provides traders with powerful tools for technical analysis across various market conditions and trading styles.
