# RSI Strategy

# Get configuration parameters
period = config.get("period", 14)
oversold = config.get("oversold", 30)
overbought = config.get("overbought", 70)
position_size = config.get("position_size", 0.01)

# Calculate RSI
rsi_values = rsi(close, period)

# Get current values
current_rsi = rsi_values[-1] if rsi_values else 50
current_price = close[-1] if close else 0

# Initialize signal
action = "hold"
quantity = 0.0
price = current_price
reason = "RSI in neutral zone"

# Check for RSI signals
if len(close) >= period:
    if current_rsi < oversold:
        action = "buy"
        quantity = position_size
        reason = f"RSI oversold at {current_rsi:.2f} (< {oversold})"
        log(f"BUY signal: {reason}")
    elif current_rsi > overbought:
        action = "sell"
        quantity = position_size
        reason = f"RSI overbought at {current_rsi:.2f} (> {overbought})"
        log(f"SELL signal: {reason}")

# Return the signal
