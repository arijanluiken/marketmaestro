# Support and Resistance Strategy
# Uses Pivot Points and Fibonacci levels for trading decisions
# Updated to use thread-safe state management

# Strategy settings configuration
def settings():
    return {
        "interval": "15m",  # Default kline interval - can be overridden in config
        "lookback_period": 20,
        "position_size": 0.01
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
lookback_period = params["lookback_period"]  # Default values, will be overridden by config
position_size = params["position_size"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "lookback_period": get_config("lookback_period", params["lookback_period"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])

def on_kline(kline):
    """Called when a new kline is received"""
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
    max_needed = 100  # Keep 100 klines for calculations
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Check if we have enough data
    if len(klines) < 20:
        return {"action": "hold", "reason": "not enough data points"}
    
    # Extract price arrays from internal buffer
    highs = [k["high"] for k in klines]
    lows = [k["low"] for k in klines]
    closes = [k["close"] for k in klines]
    
    # Calculate indicators
    
    # 1. Pivot Points for support/resistance levels
    pivot_result = pivot_points(highs, lows, closes)
    pivot = pivot_result["pivot"]
    r1 = pivot_result["r1"] 
    r2 = pivot_result["r2"]
    s1 = pivot_result["s1"]
    s2 = pivot_result["s2"]
    
    # 2. Find recent swing high and low for Fibonacci
    recent_high = 0
    recent_low = float('inf')
    lookback = min(20, len(highs))
    
    for i in range(len(highs) - lookback, len(highs)):
        if highs[i] > recent_high:
            recent_high = highs[i]
        if lows[i] < recent_low:
            recent_low = lows[i]
    
    # Calculate Fibonacci retracement levels
    fib_levels = fibonacci(recent_high, recent_low)
    fib_236 = fib_levels["23.6"]
    fib_382 = fib_levels["38.2"]
    fib_500 = fib_levels["50.0"]
    fib_618 = fib_levels["61.8"]
    
    # 3. Simple Moving Averages for trend context
    sma_20 = sma(closes, 20)
    sma_50 = sma(closes, 50) if len(closes) >= 50 else None
    
    # 4. RSI for momentum
    rsi_14 = rsi(closes, 14)
    
    # Get current values
    current_price = closes[-1]
    current_pivot = pivot[-1] if pivot[-1] else 0
    current_r1 = r1[-1] if r1[-1] else 0
    current_r2 = r2[-1] if r2[-1] else 0
    current_s1 = s1[-1] if s1[-1] else 0
    current_s2 = s2[-1] if s2[-1] else 0
    current_sma20 = sma_20[-1] if sma_20[-1] else 0
    current_sma50 = sma_50[-1] if sma_50 and sma_50[-1] else 0
    current_rsi = rsi_14[-1] if rsi_14[-1] else 50
    
    # Determine trend context
    trend = "neutral"
    if current_sma50 > 0:
        if current_sma20 > current_sma50:
            trend = "bullish"
        elif current_sma20 < current_sma50:
            trend = "bearish"
    
    # Check proximity to key levels (within 0.5% tolerance)
    tolerance = current_price * 0.005
    
    def near_level(price, level):
        return abs(price - level) <= tolerance
    
    # Strategy Logic
    position_size = cfg["position_size"]
    
    # Buy conditions - price near support in uptrend
    if trend == "bullish" and current_rsi < 70:
        # Near pivot support
        if near_level(current_price, current_pivot) and current_price > current_pivot:
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Price bouncing off pivot support at " + str(round(current_pivot, 2)) + " in bullish trend. RSI: " + str(round(current_rsi, 1))
            }
        
        # Near S1 support
        if near_level(current_price, current_s1) and current_price > current_s1:
            return {
                "action": "buy", 
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Price bouncing off S1 support at " + str(round(current_s1, 2)) + " in bullish trend. RSI: " + str(round(current_rsi, 1))
            }
        
        # Near Fibonacci support levels
        if near_level(current_price, fib_618) and current_price > fib_618:
            return {
                "action": "buy",
                "quantity": position_size, 
                "price": current_price,
                "type": "market",
                "reason": "Price bouncing off 61.8% Fibonacci level at " + str(round(fib_618, 2)) + " in bullish trend. RSI: " + str(round(current_rsi, 1))
            }
        
        if near_level(current_price, fib_500) and current_price > fib_500:
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price, 
                "type": "market",
                "reason": "Price bouncing off 50% Fibonacci level at " + str(round(fib_500, 2)) + " in bullish trend. RSI: " + str(round(current_rsi, 1))
            }
    
    # Sell conditions - price near resistance in downtrend
    if trend == "bearish" and current_rsi > 30:
        # Near pivot resistance
        if near_level(current_price, current_pivot) and current_price < current_pivot:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market", 
                "reason": "Price rejected at pivot resistance at " + str(round(current_pivot, 2)) + " in bearish trend. RSI: " + str(round(current_rsi, 1))
            }
        
        # Near R1 resistance
        if near_level(current_price, current_r1) and current_price < current_r1:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Price rejected at R1 resistance at " + str(round(current_r1, 2)) + " in bearish trend. RSI: " + str(round(current_rsi, 1))
            }
        
        # Near Fibonacci resistance levels
        if near_level(current_price, fib_382) and current_price < fib_382:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Price rejected at 38.2% Fibonacci level at " + str(round(fib_382, 2)) + " in bearish trend. RSI: " + str(round(current_rsi, 1))
            }
        
        if near_level(current_price, fib_236) and current_price < fib_236:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Price rejected at 23.6% Fibonacci level at " + str(round(fib_236, 2)) + " in bearish trend. RSI: " + str(round(current_rsi, 1))
            }
    
    # Strong breakout conditions
    if current_rsi > 60 and current_price > current_r2:
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": "Strong breakout above R2 resistance at " + str(round(current_r2, 2)) + ". RSI: " + str(round(current_rsi, 1))
        }
    
    if current_rsi < 40 and current_price < current_s2:
        return {
            "action": "sell", 
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": "Strong breakdown below S2 support at " + str(round(current_s2, 2)) + ". RSI: " + str(round(current_rsi, 1))
        }
    
    # Default hold
    return {
        "action": "hold",
        "reason": "No clear setup. Price: " + str(round(current_price, 2)) + ", Trend: " + trend + ", Pivot: " + str(round(current_pivot, 2)) + ", RSI: " + str(round(current_rsi, 1))
    }

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {"action": "hold", "reason": "No ticker signal"}
