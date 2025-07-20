# SuperTrend Strategy with MACD and RSI confirmation
# Uses thread-safe state management

# Strategy settings configuration
def settings():
    return {
        "interval": "15m",  # Default kline interval - can be overridden in config
        "supertrend_period": 10,
        "supertrend_multiplier": 3.0,
        "macd_fast": 12,
        "macd_slow": 26,
        "macd_signal": 9,
        "rsi_period": 14,
        "position_size": 0.01
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
supertrend_period = params["supertrend_period"]  # Default values, will be overridden by config
supertrend_multiplier = params["supertrend_multiplier"]
macd_fast = params["macd_fast"]
macd_slow = params["macd_slow"]
macd_signal = params["macd_signal"]
rsi_period = params["rsi_period"]
position_size = params["position_size"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "supertrend_period": get_config("supertrend_period", params["supertrend_period"]),
        "supertrend_multiplier": get_config("supertrend_multiplier", params["supertrend_multiplier"]),
        "macd_fast": get_config("macd_fast", params["macd_fast"]),
        "macd_slow": get_config("macd_slow", params["macd_slow"]),
        "macd_signal": get_config("macd_signal", params["macd_signal"]),
        "rsi_period": get_config("rsi_period", params["rsi_period"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("position", 0)
        set_state("entry_price", 0)
        set_state("stop_loss", 0)
        set_state("klines", [])

def on_kline(kline):
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    
    # Add new kline to internal buffer
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only what we need for calculations (prevent memory buildup)
    max_needed = max(cfg["macd_slow"] + cfg["macd_signal"], cfg["supertrend_period"], cfg["rsi_period"]) + 50
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Check if we have enough data
    min_periods = max(cfg["macd_slow"] + cfg["macd_signal"], cfg["supertrend_period"], cfg["rsi_period"])
    if len(klines) < min_periods + 10:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract price arrays from internal buffer
    closes = [k["close"] for k in klines]
    highs = [k["high"] for k in klines]
    lows = [k["low"] for k in klines]
    
    # Calculate indicators
    st = supertrend(highs, lows, closes, cfg["supertrend_period"], cfg["supertrend_multiplier"])
    macd_data = macd(closes, cfg["macd_fast"], cfg["macd_slow"], cfg["macd_signal"])
    rsi_values = rsi(closes, cfg["rsi_period"])
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
    
    # Get current position state
    position = get_state("position", 0)
    entry_price = get_state("entry_price", 0)
    stop_loss = get_state("stop_loss", 0)
    
    # Trading logic
    if position == 0:
        if current_trend == True and current_histogram > 0 and current_rsi > 50:
            set_state("position", 1)
            set_state("entry_price", current_price)
            set_state("stop_loss", current_price - 2 * current_atr)
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": "Supertrend buy signal with MACD and RSI confirmation"
            }
    elif position == 1:
        # Update trailing stop
        new_stop = current_price - 2 * current_atr
        if new_stop > stop_loss:
            set_state("stop_loss", new_stop)
        
        # Check stop loss
        if current_price < stop_loss:
            set_state("position", 0)
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": "Trailing stop hit"
            }
        # Check Supertrend sell signal
        elif current_trend == False:
            set_state("position", 0)
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
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