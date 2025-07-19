# Support and Resistance Strategy
# Uses Pivot Points and Fibonacci levels for trading decisions

def get_signal(data):
    """
    Strategy using pivot points and Fibonacci retracements
    """
    
    # Get price data
    high = data.get("high")
    low = data.get("low")
    close = data.get("close")
    
    if not high or not low or not close:
        return {"action": "hold", "reason": "insufficient data"}
    
    if len(close) < 20:
        return {"action": "hold", "reason": "not enough data points"}
    
    # Calculate indicators
    
    # 1. Pivot Points for support/resistance levels
    pivot_result = pivot_points(high, low, close)
    pivot = pivot_result["pivot"]
    r1 = pivot_result["r1"] 
    r2 = pivot_result["r2"]
    s1 = pivot_result["s1"]
    s2 = pivot_result["s2"]
    
    # 2. Find recent swing high and low for Fibonacci
    recent_high = 0
    recent_low = float('inf')
    lookback = min(20, len(high))
    
    for i in range(len(high) - lookback, len(high)):
        if high[i] > recent_high:
            recent_high = high[i]
        if low[i] < recent_low:
            recent_low = low[i]
    
    # Calculate Fibonacci retracement levels
    fib_levels = fibonacci(recent_high, recent_low)
    fib_236 = fib_levels["23.6"]
    fib_382 = fib_levels["38.2"]
    fib_500 = fib_levels["50.0"]
    fib_618 = fib_levels["61.8"]
    
    # 3. Simple Moving Averages for trend context
    sma_20 = sma(close, 20)
    sma_50 = sma(close, 50) if len(close) >= 50 else None
    
    # 4. RSI for momentum
    rsi_14 = rsi(close, 14)
    
    # Get current values
    current_price = close[-1]
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
    
    # Buy conditions - price near support in uptrend
    if trend == "bullish" and current_rsi < 70:
        # Near pivot support
        if near_level(current_price, current_pivot) and current_price > current_pivot:
            return {
                "action": "buy",
                "quantity": 0.1,
                "price": current_price,
                "type": "market",
                "reason": f"Price bouncing off pivot support at {current_pivot:.2f} in bullish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near S1 support
        if near_level(current_price, current_s1) and current_price > current_s1:
            return {
                "action": "buy", 
                "quantity": 0.1,
                "price": current_price,
                "type": "market",
                "reason": f"Price bouncing off S1 support at {current_s1:.2f} in bullish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near Fibonacci support levels
        if near_level(current_price, fib_618) and current_price > fib_618:
            return {
                "action": "buy",
                "quantity": 0.1, 
                "price": current_price,
                "type": "market",
                "reason": f"Price bouncing off 61.8% Fibonacci level at {fib_618:.2f} in bullish trend. RSI: {current_rsi:.1f}"
            }
        
        if near_level(current_price, fib_500) and current_price > fib_500:
            return {
                "action": "buy",
                "quantity": 0.1,
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
                "quantity": 0.1,
                "price": current_price,
                "type": "market", 
                "reason": f"Price rejected at pivot resistance at {current_pivot:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near R1 resistance
        if near_level(current_price, current_r1) and current_price < current_r1:
            return {
                "action": "sell",
                "quantity": 0.1,
                "price": current_price,
                "type": "market",
                "reason": f"Price rejected at R1 resistance at {current_r1:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
        
        # Near Fibonacci resistance levels
        if near_level(current_price, fib_382) and current_price < fib_382:
            return {
                "action": "sell",
                "quantity": 0.1,
                "price": current_price,
                "type": "market",
                "reason": f"Price rejected at 38.2% Fibonacci level at {fib_382:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
        
        if near_level(current_price, fib_236) and current_price < fib_236:
            return {
                "action": "sell",
                "quantity": 0.1,
                "price": current_price,
                "type": "market",
                "reason": f"Price rejected at 23.6% Fibonacci level at {fib_236:.2f} in bearish trend. RSI: {current_rsi:.1f}"
            }
    
    # Strong breakout conditions
    if current_rsi > 60 and current_price > current_r2:
        return {
            "action": "buy",
            "quantity": 0.1,
            "price": current_price,
            "type": "market",
            "reason": f"Strong breakout above R2 resistance at {current_r2:.2f}. RSI: {current_rsi:.1f}"
        }
    
    if current_rsi < 40 and current_price < current_s2:
        return {
            "action": "sell", 
            "quantity": 0.1,
            "price": current_price,
            "type": "market",
            "reason": f"Strong breakdown below S2 support at {current_s2:.2f}. RSI: {current_rsi:.1f}"
        }
    
    # Default hold
    return {
        "action": "hold",
        "reason": f"No clear setup. Price: {current_price:.2f}, Trend: {trend}, Pivot: {current_pivot:.2f}, RSI: {current_rsi:.1f}"
    }

# Configuration
config = {
    "symbol": "BTCUSDT", 
    "interval": "15m",
    "description": "Support and resistance strategy using pivot points and Fibonacci retracements"
}
