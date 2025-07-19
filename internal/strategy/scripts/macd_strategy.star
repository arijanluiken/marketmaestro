# MACD Strategy

# Get configuration parameters
fast_period = config.get("fast_period", 12)
slow_period = config.get("slow_period", 26)
signal_period = config.get("signal_period", 9)
position_size = config.get("position_size", 0.01)

# Calculate MACD
macd_result = macd(close, fast_period, slow_period, signal_period)
macd_line = macd_result["macd"]
signal_line = macd_result["signal"]
histogram = macd_result["histogram"]

# Get current values
current_macd = macd_line[-1] if macd_line else 0
current_signal = signal_line[-1] if signal_line else 0
current_histogram = histogram[-1] if histogram else 0
current_price = close[-1] if close else 0

# Initialize signal
action = "hold"
quantity = 0.0
price = current_price
reason = "No MACD signal"

# Check for MACD signals
if len(close) >= slow_period + signal_period:
    # Check for bullish crossover (MACD crosses above signal)
    macd_crossover = crossover(macd_line, signal_line)
    macd_crossunder = crossunder(macd_line, signal_line)
    
    if macd_crossover and macd_crossover[-1] and current_macd < 0:
        action = "buy"
        quantity = position_size
        reason = f"MACD bullish crossover below zero line"
        log(f"BUY signal: {reason}")
    
    # Check for bearish crossover (MACD crosses below signal)
    elif macd_crossunder and macd_crossunder[-1] and current_macd > 0:
        action = "sell"
        quantity = position_size
        reason = f"MACD bearish crossover above zero line"
        log(f"SELL signal: {reason}")

# Return the signal
