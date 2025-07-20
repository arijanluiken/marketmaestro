# RSI Strategy

# Strategy settings configuration
def settings():
    return {
        "interval": "15m",  # Default kline interval - can be overridden in config
        "period": 14,
        "oversold": 30,
        "overbought": 70,
        "position_size": 0.01
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
period = params["period"]  # Default values, will be overridden by config
oversold = params["oversold"]
overbought = params["overbought"]
position_size = params["position_size"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "period": get_config("period", params["period"]),
        "oversold": get_config("oversold", params["oversold"]),
        "overbought": get_config("overbought", params["overbought"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("rsi_values", [])
        set_state("klines", [])
        set_state("last_signal", "hold")

def on_kline(kline):
    """Called when a new kline is received"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    period = cfg["period"]
    oversold = cfg["oversold"]
    overbought = cfg["overbought"]
    position_size = cfg["position_size"]
    
    # Get current state
    klines = get_state("klines", [])
    
    # Add new kline to our buffer
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only the klines we need
    max_needed = period + 20
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Extract close prices
    closes = [k["close"] for k in klines]
    
    # Calculate RSI
    if len(closes) >= period:
        rsi_values = rsi(closes, period)
        set_state("rsi_values", rsi_values)
    
    # Check for trading signals
    return check_signals(closes, oversold, overbought, position_size)

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

def check_signals(closes, oversold, overbought, position_size):
    """Check for RSI overbought/oversold signals"""
    rsi_values = get_state("rsi_values", [])
    last_signal = get_state("last_signal", "hold")
    
    if len(rsi_values) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market",
            "reason": "Insufficient RSI data"
        }
    
    current_rsi = rsi_values[-1]
    previous_rsi = rsi_values[-2]
    current_price = closes[-1]
    
    # RSI oversold bounce: RSI crosses above oversold level
    if previous_rsi <= oversold and current_rsi > oversold and last_signal != "buy":
        set_state("last_signal", "buy")
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": "RSI oversold bounce: RSI (" + str(round(current_rsi, 2)) + ") crossed above " + str(oversold)
        }
    
    # RSI overbought reversal: RSI crosses below overbought level
    elif previous_rsi >= overbought and current_rsi < overbought and last_signal != "sell":
        set_state("last_signal", "sell")
        return {
            "action": "sell",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": "RSI overbought reversal: RSI (" + str(round(current_rsi, 2)) + ") crossed below " + str(overbought)
        }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No RSI signal. Current RSI: " + str(round(current_rsi, 2)) + " (oversold: " + str(oversold) + ", overbought: " + str(overbought) + ")"
    }
