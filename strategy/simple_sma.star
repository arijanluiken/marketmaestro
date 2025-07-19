# Simple Moving Average Crossover Strategy

# Strategy settings configuration
def settings():
    return {
        "interval": "1m",  # Kline interval for this strategy
        "short_period": 10,
        "long_period": 20,
        "position_size": 0.01
    }

# Strategy state
state = {
    "short_ma": [],
    "long_ma": [],
    "klines": [],
    "last_signal": "hold"
}

# Initialize strategy parameters
params = settings()
short_period = config.get("short_period", params["short_period"])
long_period = config.get("long_period", params["long_period"])
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
    max_needed = max(short_period, long_period) + 10
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Extract close prices
    closes = [k["close"] for k in state["klines"]]
    
    # Calculate moving averages
    if len(closes) >= short_period:
        state["short_ma"] = sma(closes, short_period)
    
    if len(closes) >= long_period:
        state["long_ma"] = sma(closes, long_period)
    
    # Check for trading signals
    return check_signals(closes)

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    # Could use spread analysis or other orderbook-based logic
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
    # Could use volume or price change analysis
    # For now, just return hold
    return {
        "action": "hold", 
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No ticker signal"
    }

def check_signals(closes):
    """Check for crossover signals"""
    if len(state["short_ma"]) < 2 or len(state["long_ma"]) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market", 
            "reason": "Insufficient data for signal"
        }
    
    # Get current and previous values
    current_short = state["short_ma"][-1]
    current_long = state["long_ma"][-1]
    prev_short = state["short_ma"][-2]
    prev_long = state["long_ma"][-2]
    current_price = closes[-1]
    
    # Check for bullish crossover (short MA crosses above long MA)
    if prev_short <= prev_long and current_short > current_long:
        reason = "Short MA (" + str(round(current_short, 2)) + ") crossed above Long MA (" + str(round(current_long, 2)) + ")"
        log("BUY signal: " + reason)
        state["last_signal"] = "buy"
        
        # Place market buy order with trailing stop loss
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason,
            # Add trailing stop configuration
            "stop_loss": {
                "type": "trailing_stop",
                "trail_percent": 2.0,  # 2% trailing stop
                "side": "sell"
            }
        }
    
    # Check for bearish crossover (short MA crosses below long MA)
    elif prev_short >= prev_long and current_short < current_long:
        reason = "Short MA (" + str(round(current_short, 2)) + ") crossed below Long MA (" + str(round(current_long, 2)) + ")"
        log("SELL signal: " + reason)
        state["last_signal"] = "sell"
        
        # Place market sell order with stop loss
        stop_price = current_price * 1.02  # 2% stop loss above current price for short
        return {
            "action": "sell",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason,
            # Add stop loss configuration
            "stop_loss": {
                "type": "stop_market",
                "stop_price": stop_price,
                "side": "buy"
            }
        }
    
    # No signal
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No crossover detected"
    }
