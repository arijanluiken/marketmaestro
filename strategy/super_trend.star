# SuperTrend Strategy with MACD and RSI confirmation
# Uses callback data instead of global variables

state = {
    "position": 0,
    "entry_price": 0,
    "stop_loss": 0,
    "klines": []  # Internal kline buffer
}

def settings():
    return {
        "interval": "15m",
        "supertrend_period": 10,
        "supertrend_multiplier": 3.0,
        "macd_fast": 12,
        "macd_slow": 26,
        "macd_signal": 9,
        "rsi_period": 14,
        "position_size": 0.01
    }

def on_kline(kline):
    # Get configuration
    supertrend_period = config.get("supertrend_period", 10)
    supertrend_multiplier = config.get("supertrend_multiplier", 3.0)
    macd_fast = config.get("macd_fast", 12)
    macd_slow = config.get("macd_slow", 26)
    macd_signal = config.get("macd_signal", 9)
    rsi_period = config.get("rsi_period", 14)
    position_size = config.get("position_size", 0.01)
    
    # Add new kline to internal buffer
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only what we need for calculations (prevent memory buildup)
    max_needed = max(macd_slow + macd_signal, supertrend_period, rsi_period) + 50
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Check if we have enough data
    min_periods = max(macd_slow + macd_signal, supertrend_period, rsi_period)
    if len(state["klines"]) < min_periods + 10:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract price arrays from internal buffer
    closes = [k["close"] for k in state["klines"]]
    highs = [k["high"] for k in state["klines"]]
    lows = [k["low"] for k in state["klines"]]
    
    # Calculate indicators
    st = supertrend(highs, lows, closes, supertrend_period, supertrend_multiplier)
    macd_data = macd(closes, macd_fast, macd_slow, macd_signal)
    rsi_values = rsi(closes, rsi_period)
    atr_values = atr(highs, lows, closes, 14)
    
    # Get latest values
    current_trend = st["trend"][-1]
    current_histogram = macd_data["histogram"][-1]
    current_rsi = rsi_values[-1]
    current_atr = atr_values[-1]
    current_price = closes[-1]
    
    # Check for invalid indicator values
    if math.isnan(current_histogram) or math.isnan(current_rsi) or math.isnan(current_atr):
        return {"action": "hold", "reason": "Invalid indicator values"}
    
    # Trading logic
    if state["position"] == 0:
        if current_trend == True and current_histogram > 0 and current_rsi > 50:
            state["position"] = 1
            state["entry_price"] = current_price
            state["stop_loss"] = current_price - 2 * current_atr
            return {
                "action": "buy",
                "quantity": position_size,
                "type": "market",
                "reason": "Supertrend buy signal with MACD and RSI confirmation"
            }
    elif state["position"] == 1:
        # Update trailing stop
        new_stop = current_price - 2 * current_atr
        if new_stop > state["stop_loss"]:
            state["stop_loss"] = new_stop
        
        # Check stop loss
        if current_price < state["stop_loss"]:
            state["position"] = 0
            return {
                "action": "sell",
                "quantity": position_size,
                "type": "market",
                "reason": "Trailing stop hit"
            }
        # Check Supertrend sell signal
        elif current_trend == False:
            state["position"] = 0
            return {
                "action": "sell",
                "quantity": position_size,
                "type": "market",
                "reason": "Supertrend sell signal"
            }
    
    return {"action": "hold"}

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {"action": "hold", "reason": "No ticker signal"}