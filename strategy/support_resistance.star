# Support and Resistance Strategy
# Uses Pivot Points and Fibonacci levels for trading decisions
# Updated to use callback-provided data instead of global variables

state = {
    "klines": []
}

def settings():
    return {
        "interval": "15m",
        "lookback_period": 20,
        "position_size": 0.01
    }

def on_kline(kline):
    """Called when a new kline is received"""
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
    max_needed = 100  # Keep 100 klines for calculations
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Check if we have enough data
    if len(state["klines"]) < 20:
        return {"action": "hold", "reason": "not enough data points"}
    
    # Extract price arrays from internal buffer
    highs = [k["high"] for k in state["klines"]]
    lows = [k["low"] for k in state["klines"]]
    closes = [k["close"] for k in state["klines"]]
    
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
    position_size = config.get("position_size", 0.01)
    
    # Buy conditions - price near support in uptrend
    if trend == "bullish" and current_rsi < 70:
        # Near pivot support
        if near_level(current_price, current_pivot) and current_price > current_pivot:
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": f"Price bouncing off pivot support at {current_pivot:.2f} in bullish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near S1 support
        if near_level(current_price, current_s1) and current_price > current_s1:
            return {
                "action": "buy", 
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": f"Price bouncing off S1 support at {current_s1:.2f} in bullish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near Fibonacci support levels
        if near_level(current_price, fib_618) and current_price > fib_618:
            return {
                "action": "buy",
                "quantity": position_size, 
                "price": current_price,
                "type": "market",
                "reason": f"Price bouncing off 61.8% Fibonacci level at {fib_618:.2f} in bullish trend. RSI: {current_rsi:.1f}"
            }
        
        if near_level(current_price, fib_500) and current_price > fib_500:
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price, 
                "type": "market",
                "reason": f"Price bouncing off 50% Fibonacci level at {fib_500:.2f} in bullish trend. RSI: {current_rsi:.1f}"
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
                "reason": f"Price rejected at pivot resistance at {current_pivot:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near R1 resistance
        if near_level(current_price, current_r1) and current_price < current_r1:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": f"Price rejected at R1 resistance at {current_r1:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near Fibonacci resistance levels
        if near_level(current_price, fib_382) and current_price < fib_382:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": f"Price rejected at 38.2% Fibonacci level at {fib_382:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
        
        if near_level(current_price, fib_236) and current_price < fib_236:
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": f"Price rejected at 23.6% Fibonacci level at {fib_236:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
    
    # Strong breakout conditions
    if current_rsi > 60 and current_price > current_r2:
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": f"Strong breakout above R2 resistance at {current_r2:.2f}. RSI: {current_rsi:.1f}"
        }
    
    if current_rsi < 40 and current_price < current_s2:
        return {
            "action": "sell", 
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": f"Strong breakdown below S2 support at {current_s2:.2f}. RSI: {current_rsi:.1f}"
        }
    
    # Default hold
    return {
        "action": "hold",
        "reason": f"No clear setup. Price: {current_price:.2f}, Trend: {trend}, Pivot: {current_pivot:.2f}, RSI: {current_rsi:.1f}"
    }

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {"action": "hold", "reason": "No ticker signal"}
