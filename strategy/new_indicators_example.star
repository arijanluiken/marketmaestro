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
    # Ensure we have enough data
    if len(close) < 50:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Calculate Hull Moving Average for trend direction
    hma = hull_ma(close, config.get("hma_period", 21))
    current_hma = hma[-1]
    prev_hma = hma[-2]
    
    # Calculate Chande Momentum Oscillator for momentum
    cmo = cmo(close, config.get("cmo_period", 14))
    current_cmo = cmo[-1]
    
    # Calculate Schaff Trend Cycle for overbought/oversold
    stc = stc(close, config.get("stc_fast", 12), config.get("stc_slow", 26), 10, 0.5)
    current_stc = stc[-1]
    
    # Calculate Balance of Power for buying/selling pressure
    bop_values = bop(open, high, low, close)
    current_bop = bop_values[-1]
    
    current_price = close[-1]
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