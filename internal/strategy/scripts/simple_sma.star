# Simple Moving Average Crossover Strategy

# Get configuration parameters
short_period = config.get("short_period", 10)
long_period = config.get("long_period", 20)
position_size = config.get("position_size", 0.01)

# Calculate moving averages
short_ma = sma(close, short_period)
long_ma = sma(close, long_period)

# Get current values (last in series)
current_short = short_ma[-1] if short_ma else 0
current_long = long_ma[-1] if long_ma else 0
current_price = close[-1] if close else 0

# Initialize signal
action = "hold"
quantity = 0.0
price = current_price
reason = "No signal"

# Check for crossover signals
if len(close) >= long_period:
    # Check if short MA crosses above long MA (buy signal)
    crossover_signals = crossover(short_ma, long_ma)
    crossunder_signals = crossunder(short_ma, long_ma)
    
    if crossover_signals and crossover_signals[-1]:
        action = "buy"
        quantity = position_size
        reason = f"Short MA ({current_short:.2f}) crossed above Long MA ({current_long:.2f})"
        log(f"BUY signal: {reason}")
    
    # Check if short MA crosses below long MA (sell signal)
    elif crossunder_signals and crossunder_signals[-1]:
        action = "sell"
        quantity = position_size
        reason = f"Short MA ({current_short:.2f}) crossed below Long MA ({current_long:.2f})"
        log(f"SELL signal: {reason}")

# Return the signal
# These variables will be read by the strategy engine
# action: "buy", "sell", "hold"
# quantity: amount to trade
# price: target price (0 for market orders)
# type: "market" or "limit"
# reason: explanation for the signal
