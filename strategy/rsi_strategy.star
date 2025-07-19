# RSI Strategy

# Strategy settings configuration
def settings():
    return {
        "interval": "15m",  # Kline interval for this strategy
        "period": 14,
        "oversold": 30,
        "overbought": 70,
        "position_size": 0.01
    }

# Strategy state
state = {
    "rsi_values": [],
    "klines": [],
    "last_signal": "hold"
}

# Initialize strategy parameters
params = settings()
period = config.get("period", params["period"])
oversold = config.get("oversold", params["oversold"])
overbought = config.get("overbought", params["overbought"])
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
    max_needed = period + 20
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Extract close prices
    closes = [k["close"] for k in state["klines"]]
    
    # Calculate RSI
    if len(closes) >= period:
        state["rsi_values"] = rsi(closes, period)
    
    # Check for trading signals
    return check_signals(closes)

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    # Could use spread analysis for RSI confirmation
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
    # Could use volume for RSI confirmation
    # For now, just return hold
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No ticker signal"
    }

def check_signals(closes):
    """Check for RSI signals"""
    if len(state["rsi_values"]) < 1:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market",
            "reason": "Insufficient data for RSI calculation"
        }
    
    current_rsi = state["rsi_values"][-1]
    current_price = closes[-1]
    
    # Check for oversold condition (buy signal)
    if current_rsi < oversold:
        reason = "RSI oversold at " + str(round(current_rsi, 2)) + " (< " + str(oversold) + ")"
        log("BUY signal: " + reason)
        state["last_signal"] = "buy"
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    # Check for overbought condition (sell signal)
    elif current_rsi > overbought:
        reason = "RSI overbought at " + str(round(current_rsi, 2)) + " (> " + str(overbought) + ")"
        log("SELL signal: " + reason)
        state["last_signal"] = "sell"
        return {
            "action": "sell",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    # No signal - RSI in neutral zone
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "RSI in neutral zone (" + str(round(current_rsi, 2)) + ")"
    }

# Return the signal
