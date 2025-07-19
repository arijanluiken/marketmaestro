# MACD Strategy

# Strategy settings configuration
def settings():
    return {
        "interval": "5m",  # Kline interval for this strategy
        "fast_period": 12,
        "slow_period": 26,
        "signal_period": 9,
        "position_size": 0.01
    }

# Strategy state
state = {
    "macd_line": [],
    "signal_line": [],
    "histogram": [],
    "klines": [],
    "last_signal": "hold"
}

# Initialize strategy parameters
params = settings()
fast_period = config.get("fast_period", params["fast_period"])
slow_period = config.get("slow_period", params["slow_period"])
signal_period = config.get("signal_period", params["signal_period"])
position_size = config.get("position_size", params["position_size"])

def on_kline(kline):
    """Called when a new kline is received"""
    # Add new kline to our buffer
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only the klines we need
    max_needed = slow_period + signal_period + 20
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Extract close prices
    closes = [k["close"] for k in state["klines"]]
    
    # Calculate MACD
    if len(closes) >= slow_period + signal_period:
        macd_result = macd(closes, fast_period, slow_period, signal_period)
        state["macd_line"] = macd_result["macd"]
        state["signal_line"] = macd_result["signal"]
        state["histogram"] = macd_result["histogram"]
    
    # Check for trading signals
    return check_signals(closes)

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    # Could use orderbook for MACD confirmation
    # For now, just return hold
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No orderbook signal"
    }

def on_ticker(ticker):
    """Called when ticker data is received"""
    # Could use volume for MACD confirmation
    # For now, just return hold
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No ticker signal"
    }

def check_signals(closes):
    """Check for MACD signals"""
    if len(state["macd_line"]) < 2 or len(state["signal_line"]) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market",
            "reason": "Insufficient data for MACD calculation"
        }
    
    # Get current and previous values
    current_macd = state["macd_line"][-1]
    current_signal = state["signal_line"][-1]
    prev_macd = state["macd_line"][-2]
    prev_signal = state["signal_line"][-2]
    current_price = closes[-1]
    
    # Check for bullish crossover (MACD crosses above signal) below zero line
    if prev_macd <= prev_signal and current_macd > current_signal and current_macd < 0:
        reason = "MACD bullish crossover below zero line"
        log("BUY signal: " + reason)
        state["last_signal"] = "buy"
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    # Check for bearish crossover (MACD crosses below signal) above zero line
    elif prev_macd >= prev_signal and current_macd < current_signal and current_macd > 0:
        reason = "MACD bearish crossover above zero line"
        log("SELL signal: " + reason)
        state["last_signal"] = "sell"
        return {
            "action": "sell",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    # No signal
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No MACD crossover signal"
    }
