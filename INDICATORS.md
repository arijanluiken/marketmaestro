# Technical Indicators Reference

This document provides a comprehensive reference for all technical indicators available in the Mercantile strategy engine.

## Available Indicators (15 Total)

### Moving Averages

#### Simple Moving Average (SMA)
```python
sma_values = sma(prices, period)
```
- **Purpose**: Smooths price data to identify trend direction
- **Parameters**: 
  - `prices`: List of price values (typically close prices)
  - `period`: Number of periods for the average
- **Returns**: List of SMA values
- **Common Usage**: Trend identification, support/resistance levels

#### Exponential Moving Average (EMA)
```python
ema_values = ema(prices, period)
```
- **Purpose**: Moving average that gives more weight to recent prices
- **Parameters**: 
  - `prices`: List of price values
  - `period`: Number of periods for the average
- **Returns**: List of EMA values
- **Common Usage**: Faster trend detection, MACD calculation

### Momentum Oscillators

#### Relative Strength Index (RSI)
```python
rsi_values = rsi(prices, period)
```
- **Purpose**: Measures speed and magnitude of price changes
- **Parameters**: 
  - `prices`: List of close prices
  - `period`: Number of periods (typically 14)
- **Returns**: List of RSI values (0-100)
- **Common Usage**: Overbought (>70) and oversold (<30) conditions

#### Williams %R
```python
williams_r_values = williams_r(high, low, close, period)
```
- **Purpose**: Momentum oscillator comparing close to high-low range
- **Parameters**: 
  - `high`: List of high prices
  - `low`: List of low prices  
  - `close`: List of close prices
  - `period`: Number of periods (typically 14)
- **Returns**: List of Williams %R values (-100 to 0)
- **Common Usage**: Overbought (>-20) and oversold (<-80) conditions

#### Stochastic Oscillator
```python
stoch_result = stochastic(high, low, close, k_period, d_period)
current_k = stoch_result["k"][-1]
current_d = stoch_result["d"][-1]
```
- **Purpose**: Compares close price to price range over time
- **Parameters**: 
  - `high`: List of high prices
  - `low`: List of low prices
  - `close`: List of close prices
  - `k_period`: Period for %K calculation (typically 14)
  - `d_period`: Period for %D smoothing (typically 3)
- **Returns**: Dictionary with "k" and "d" arrays
- **Common Usage**: Overbought (>80) and oversold (<20) conditions

#### Rate of Change (ROC)
```python
roc_values = roc(prices, period)
```
- **Purpose**: Measures percentage change in price over time
- **Parameters**: 
  - `prices`: List of price values
  - `period`: Number of periods for comparison
- **Returns**: List of ROC percentage values
- **Common Usage**: Momentum analysis, trend strength

### Volatility Indicators

#### Average True Range (ATR)
```python
atr_values = atr(high, low, close, period)
```
- **Purpose**: Measures market volatility
- **Parameters**: 
  - `high`: List of high prices
  - `low`: List of low prices
  - `close`: List of close prices
  - `period`: Number of periods (typically 14)
- **Returns**: List of ATR values
- **Common Usage**: Stop loss calculation, position sizing

#### Bollinger Bands
```python
bb_result = bollinger(prices, period, multiplier)
upper_band = bb_result["upper"][-1]
middle_band = bb_result["middle"][-1]
lower_band = bb_result["lower"][-1]
```
- **Purpose**: Price channels based on standard deviation
- **Parameters**: 
  - `prices`: List of price values
  - `period`: Period for moving average (typically 20)
  - `multiplier`: Standard deviation multiplier (typically 2.0)
- **Returns**: Dictionary with "upper", "middle", "lower" arrays
- **Common Usage**: Volatility analysis, mean reversion

#### Standard Deviation
```python
stddev_values = stddev(prices, period)
```
- **Purpose**: Statistical measure of price volatility
- **Parameters**: 
  - `prices`: List of price values
  - `period`: Number of periods for calculation
- **Returns**: List of standard deviation values
- **Common Usage**: Volatility measurement, risk assessment

### Trend Indicators

#### MACD (Moving Average Convergence Divergence)
```python
macd_result = macd(prices, fast_period, slow_period, signal_period)
macd_line = macd_result["macd"][-1]
signal_line = macd_result["signal"][-1]
histogram = macd_result["histogram"][-1]
```
- **Purpose**: Trend-following momentum indicator
- **Parameters**: 
  - `prices`: List of price values
  - `fast_period`: Fast EMA period (typically 12)
  - `slow_period`: Slow EMA period (typically 26)
  - `signal_period`: Signal line EMA period (typically 9)
- **Returns**: Dictionary with "macd", "signal", "histogram" arrays
- **Common Usage**: Trend changes, momentum shifts

#### Commodity Channel Index (CCI)
```python
cci_values = cci(high, low, close, period)
```
- **Purpose**: Identifies cyclical trends and overbought/oversold conditions
- **Parameters**: 
  - `high`: List of high prices
  - `low`: List of low prices
  - `close`: List of close prices
  - `period`: Number of periods (typically 20)
- **Returns**: List of CCI values
- **Common Usage**: Trend identification, range detection

### Volume Indicators

#### Volume Weighted Average Price (VWAP)
```python
vwap_values = vwap(high, low, close, volume)
```
- **Purpose**: Average price weighted by volume
- **Parameters**: 
  - `high`: List of high prices
  - `low`: List of low prices
  - `close`: List of close prices
  - `volume`: List of volume values
- **Returns**: List of VWAP values
- **Common Usage**: Institutional price reference, fair value

#### Money Flow Index (MFI)
```python
mfi_values = mfi(high, low, close, volume, period)
```
- **Purpose**: Volume-weighted RSI, measures buying/selling pressure
- **Parameters**: 
  - `high`: List of high prices
  - `low`: List of low prices
  - `close`: List of close prices
  - `volume`: List of volume values
  - `period`: Number of periods (typically 14)
- **Returns**: List of MFI values (0-100)
- **Common Usage**: Overbought (>80) and oversold (<20) with volume confirmation

### Utility Functions

#### Highest/Lowest
```python
highest_values = highest(prices, period)
lowest_values = lowest(prices, period)
```
- **Purpose**: Rolling highest/lowest values over period
- **Parameters**: 
  - `prices`: List of price values
  - `period`: Number of periods to look back
- **Returns**: List of highest/lowest values
- **Common Usage**: Support/resistance levels, channel identification

#### Crossover/Crossunder
```python
crossover_signals = crossover(series1, series2)
crossunder_signals = crossunder(series1, series2)
```
- **Purpose**: Detect when one series crosses above/below another
- **Parameters**: 
  - `series1`: First data series (e.g., fast MA)
  - `series2`: Second data series (e.g., slow MA)
- **Returns**: List of boolean values indicating crossover points
- **Common Usage**: Signal generation, trend change detection

## Strategy Examples

### Basic RSI Strategy
```python
# Calculate RSI
rsi_values = rsi(close, 14)
current_rsi = rsi_values[-1]

# Generate signals
if current_rsi < 30:
    action = "buy"  # Oversold
elif current_rsi > 70:
    action = "sell"  # Overbought
```

### Multi-Indicator Confirmation
```python
# Calculate multiple indicators
rsi_values = rsi(close, 14)
williams_r_values = williams_r(high, low, close, 14)
mfi_values = mfi(high, low, close, volume, 14)

# Require multiple confirmations
bullish_signals = 0
if rsi_values[-1] < 30:
    bullish_signals += 1
if williams_r_values[-1] < -80:
    bullish_signals += 1
if mfi_values[-1] < 20:
    bullish_signals += 1

if bullish_signals >= 2:
    action = "buy"  # Multiple confirmations
```

### Volatility-Based Position Sizing
```python
# Calculate ATR for position sizing
atr_values = atr(high, low, close, 14)
current_atr = atr_values[-1]

# Risk-based position size
risk_amount = account_balance * 0.02  # 2% risk
stop_distance = current_atr * 2  # 2x ATR stop
position_size = risk_amount / stop_distance
```

## Best Practices

1. **Always check for sufficient data** before using indicators
2. **Use multiple indicators** for confirmation
3. **Consider market conditions** (trending vs. ranging)
4. **Apply proper risk management** with stop losses
5. **Backtest strategies** before live trading
6. **Use volatility indicators** for position sizing
7. **Combine different indicator types** (momentum, trend, volume)

## Notes

- All indicators return `None` or `NaN` for periods with insufficient data
- Volume indicators require volume data in addition to price data
- Oscillators typically have defined ranges (RSI: 0-100, Williams %R: -100 to 0)
- Trend indicators may not have fixed ranges
- Always handle edge cases in your strategy logic