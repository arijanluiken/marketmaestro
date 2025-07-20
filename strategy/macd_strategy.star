# MACD Strategy

# Strategy settings configuration
def settings():
    return {
        "interval": "5m",  # Default kline interval - can be overridden in config
        "fast_period": 12,
        "slow_period": 26,
        "signal_period": 9,
        "position_size": 0.01
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
fast_period = params["fast_period"]  # Default values, will be overridden by config
slow_period = params["slow_period"]
signal_period = params["signal_period"]
position_size = params["position_size"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "fast_period": get_config("fast_period", params["fast_period"]),
        "slow_period": get_config("slow_period", params["slow_period"]),
        "signal_period": get_config("signal_period", params["signal_period"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("macd_line", [])
        set_state("signal_line", [])
        set_state("histogram", [])
        set_state("klines", [])
        set_state("last_signal", "hold")

def on_kline(kline):
    """Called when a new kline is received"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    fast_period = cfg["fast_period"]
    slow_period = cfg["slow_period"]
    signal_period = cfg["signal_period"]
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
    max_needed = slow_period + signal_period + 20
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Extract close prices
    closes = [k["close"] for k in klines]
    
    # Calculate MACD
    if len(closes) >= slow_period + signal_period:
        macd_result = macd(closes, fast_period, slow_period, signal_period)
        set_state("macd_line", macd_result["macd"])
        set_state("signal_line", macd_result["signal"])
        set_state("histogram", macd_result["histogram"])
    
    # Check for trading signals
    return check_signals(closes, cfg)

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

def check_signals(closes, cfg):
    """Check for MACD signals"""
    macd_line = get_state("macd_line", [])
    signal_line = get_state("signal_line", [])
    
    if len(macd_line) < 2 or len(signal_line) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market",
            "reason": "Insufficient data for MACD calculation"
        }
    
    # Get current and previous values
    current_macd = macd_line[-1]
    current_signal = signal_line[-1]
    prev_macd = macd_line[-2]
    prev_signal = signal_line[-2]
    current_price = closes[-1]
    
    # Check for bullish crossover (MACD crosses above signal) below zero line
    if prev_macd <= prev_signal and current_macd > current_signal and current_macd < 0:
        reason = "MACD bullish crossover below zero line"
        log("BUY signal: " + reason)
        set_state("last_signal", "buy")
        return {
            "action": "buy",
            "quantity": cfg["position_size"],
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    # Check for bearish crossover (MACD crosses below signal) above zero line
    elif prev_macd >= prev_signal and current_macd < current_signal and current_macd > 0:
        reason = "MACD bearish crossover above zero line"
        log("SELL signal: " + reason)
        set_state("last_signal", "sell")
        return {
            "action": "sell",
            "quantity": cfg["position_size"],
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
