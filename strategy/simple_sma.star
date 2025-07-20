# Simple Moving Average Crossover Strategy

# Strategy settings configuration
def settings():
    return {
        "interval": "1m",  # Default kline interval - can be overridden in config
        "short_period": 10,
        "long_period": 20,
        "position_size": 0.01
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
short_period = params["short_period"]  # Default values, will be overridden by config
long_period = params["long_period"]
position_size = params["position_size"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "short_period": get_config("short_period", params["short_period"]),
        "long_period": get_config("long_period", params["long_period"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("short_ma", [])
        set_state("long_ma", [])
        set_state("klines", [])
        set_state("last_signal", "hold")

def on_kline(kline):
    """Called when a new kline is received"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    short_period = cfg["short_period"]
    long_period = cfg["long_period"]
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
    max_needed = max(short_period, long_period) + 10
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Extract close prices
    closes = [k["close"] for k in klines]
    
    # Calculate moving averages
    if len(closes) >= short_period:
        short_ma = sma(closes, short_period)
        set_state("short_ma", short_ma)
    
    if len(closes) >= long_period:
        long_ma = sma(closes, long_period)
        set_state("long_ma", long_ma)
    
    # Check for trading signals
    return check_signals(closes, position_size)

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

def check_signals(closes, position_size):
    """Check for crossover signals"""
    short_ma = get_state("short_ma", [])
    long_ma = get_state("long_ma", [])
    last_signal = get_state("last_signal", "hold")
    
    if len(short_ma) < 2 or len(long_ma) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market",
            "reason": "Insufficient data for MA crossover"
        }
    
    current_short = short_ma[-1]
    current_long = long_ma[-1]
    previous_short = short_ma[-2]
    previous_long = long_ma[-2]
    
    current_price = closes[-1]
    
    # Bullish crossover: short MA crosses above long MA
    if previous_short <= previous_long and current_short > current_long and last_signal != "buy":
        set_state("last_signal", "buy")
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": "SMA crossover: short MA (" + str(round(current_short, 2)) + ") crossed above long MA (" + str(round(current_long, 2)) + ")"
        }
    
    # Bearish crossover: short MA crosses below long MA  
    elif previous_short >= previous_long and current_short < current_long and last_signal != "sell":
        set_state("last_signal", "sell")
        return {
            "action": "sell",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": "SMA crossover: short MA (" + str(round(current_short, 2)) + ") crossed below long MA (" + str(round(current_long, 2)) + ")"
        }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No crossover signal. Short MA: " + str(round(current_short, 2)) + ", Long MA: " + str(round(current_long, 2))
    }
