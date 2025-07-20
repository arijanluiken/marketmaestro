# New Indicators Example Strategy
# Updated to use callback-provided data instead of global variables

state = {
    "klines": []
}

def settings():
    return {
        "interval": "5m",
        "hma_period": 21,
        "cmo_period": 14,
        "stc_fast": 12,
        "stc_slow": 26,
        "position_size": 0.01
    }

def on_kline(kline):
    # Add new kline to internal buffer
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only what we need for calculations
    max_needed = 100
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Ensure we have enough data
    if len(state["klines"]) < 50:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract price arrays from internal buffer
    opens = [k["open"] for k in state["klines"]]
    highs = [k["high"] for k in state["klines"]]
    lows = [k["low"] for k in state["klines"]]
    closes = [k["close"] for k in state["klines"]]
    
    # Calculate Hull Moving Average for trend direction
    hma = hull_ma(closes, config.get("hma_period", 21))
    current_hma = hma[-1]
    prev_hma = hma[-2]
    
    # Calculate Chande Momentum Oscillator for momentum
    cmo = cmo(closes, config.get("cmo_period", 14))
    current_cmo = cmo[-1]
    
    # Calculate Schaff Trend Cycle for overbought/oversold
    stc = stc(closes, config.get("stc_fast", 12), config.get("stc_slow", 26), 10, 0.5)
    current_stc = stc[-1]
    
    # Calculate Balance of Power for buying/selling pressure
    bop_values = bop(opens, highs, lows, closes)
    current_bop = bop_values[-1]
    
    current_price = closes[-1]
    position_size = config.get("position_size", 0.01)
    
    # HMA trend detection
    hma_uptrend = current_hma > prev_hma
    
    # Entry conditions - HMA uptrend + strong momentum + not overbought + buying pressure
    if (hma_uptrend and 
        current_cmo > 20 and 
        current_stc < 75 and 
        current_bop > 0.1):
        
        return {
            "action": "buy",
            "quantity": position_size,
            "type": "market",
            "reason": f"HMA uptrend, CMO: {round(current_cmo, 1)}, STC: {round(current_stc, 1)}, BOP: {round(current_bop, 2)}"
        }
    
    # Exit conditions - HMA downtrend or weak momentum or overbought
    elif (not hma_uptrend or 
          current_cmo < -20 or 
          current_stc > 80 or 
          current_bop < -0.1):
        
        return {
            "action": "sell",
            "quantity": position_size,
            "type": "market",
            "reason": f"Exit signal - HMA trend: {hma_uptrend}, CMO: {round(current_cmo, 1)}, STC: {round(current_stc, 1)}"
        }
    
    return {
        "action": "hold",
        "reason": f"Waiting for signals - CMO: {round(current_cmo, 1)}, STC: {round(current_stc, 1)}"
    }

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {"action": "hold", "reason": "No ticker signal"}