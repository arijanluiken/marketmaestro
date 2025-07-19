# Advanced Indicators Demo Strategy
# Demonstrates the new technical indicators available in the strategy engine

def get_signal(data):
    """
    Demo strategy showcasing advanced indicators
    """
    
    # Get price data
    high = data.get("high")
    low = data.get("low")
    close = data.get("close")
    volume = data.get("volume")
    
    if not high or not low or not close or not volume:
        return {"action": "hold", "reason": "insufficient data"}
    
    if len(close) < 50:
        return {"action": "hold", "reason": "not enough data points"}
    
    # Calculate various indicators
    
    # 1. ADX for trend strength
    adx_result = adx(high, low, close, 14)
    adx_line = adx_result["adx"]
    plus_di = adx_result["plus_di"]
    minus_di = adx_result["minus_di"]
    
    # 2. Parabolic SAR for trend direction
    psar = parabolic_sar(high, low, step=0.02, max_step=0.2)
    
    # 3. Keltner Channels for volatility
    keltner_result = keltner(high, low, close, 20, multiplier=2.0)
    keltner_upper = keltner_result["upper"]
    keltner_middle = keltner_result["middle"]
    keltner_lower = keltner_result["lower"]
    
    # 4. On-Balance Volume for volume analysis
    obv_line = obv(close, volume)
    
    # 5. Aroon for momentum
    aroon_result = aroon(high, low, 14)
    aroon_up = aroon_result["aroon_up"]
    aroon_down = aroon_result["aroon_down"]
    
    # 6. Ichimoku Cloud components
    ichimoku_result = ichimoku(high, low, close, 
                              conversion_period=9, 
                              base_period=26, 
                              span_b_period=52, 
                              displacement=26)
    tenkan_sen = ichimoku_result["tenkan_sen"]
    kijun_sen = ichimoku_result["kijun_sen"]
    
    # Get current values (last data point)
    current_price = close[-1]
    current_adx = adx_line[-1] if adx_line[-1] else 0
    current_psar = psar[-1] if psar[-1] else 0
    current_plus_di = plus_di[-1] if plus_di[-1] else 0
    current_minus_di = minus_di[-1] if minus_di[-1] else 0
    current_keltner_upper = keltner_upper[-1] if keltner_upper[-1] else 0
    current_keltner_lower = keltner_lower[-1] if keltner_lower[-1] else 0
    current_aroon_up = aroon_up[-1] if aroon_up[-1] else 0
    current_aroon_down = aroon_down[-1] if aroon_down[-1] else 0
    current_tenkan = tenkan_sen[-1] if tenkan_sen[-1] else 0
    current_kijun = kijun_sen[-1] if kijun_sen[-1] else 0
    
    # Calculate OBV trend (simple comparison with previous values)
    obv_trend = "neutral"
    if len(obv_line) >= 5:
        recent_obv = obv_line[-5:]
        if recent_obv[-1] > recent_obv[0]:
            obv_trend = "up"
        elif recent_obv[-1] < recent_obv[0]:
            obv_trend = "down"
    
    # Strategy Logic
    
    # Strong bullish conditions
    bullish_conditions = 0
    bearish_conditions = 0
    
    # ADX trend strength (above 25 indicates strong trend)
    if current_adx > 25:
        if current_plus_di > current_minus_di:
            bullish_conditions += 2
        else:
            bearish_conditions += 2
    
    # Parabolic SAR trend direction
    if current_price > current_psar:
        bullish_conditions += 1
    else:
        bearish_conditions += 1
    
    # Keltner Channel position
    if current_price > current_keltner_upper:
        bullish_conditions += 1
    elif current_price < current_keltner_lower:
        bearish_conditions += 1
    
    # Aroon momentum
    if current_aroon_up > 70 and current_aroon_down < 30:
        bullish_conditions += 2
    elif current_aroon_down > 70 and current_aroon_up < 30:
        bearish_conditions += 2
    
    # Ichimoku trend (simple version)
    if current_tenkan > current_kijun:
        bullish_conditions += 1
    else:
        bearish_conditions += 1
    
    # OBV trend
    if obv_trend == "up":
        bullish_conditions += 1
    elif obv_trend == "down":
        bearish_conditions += 1
    
    # Decision making
    if bullish_conditions >= 6:
        action = "buy"
        reason = f"Strong bullish signals: {bullish_conditions}/8 conditions met. ADX: {current_adx:.2f}, Price vs PSAR: {'above' if current_price > current_psar else 'below'}, OBV trend: {obv_trend}"
    elif bearish_conditions >= 6:
        action = "sell"
        reason = f"Strong bearish signals: {bearish_conditions}/8 conditions met. ADX: {current_adx:.2f}, Price vs PSAR: {'above' if current_price > current_psar else 'below'}, OBV trend: {obv_trend}"
    else:
        action = "hold"
        reason = f"Mixed signals - Bullish: {bullish_conditions}, Bearish: {bearish_conditions}. ADX: {current_adx:.2f} (trend strength)"
    
    return {
        "action": action,
        "quantity": 0.1,
        "price": current_price,
        "type": "market",
        "reason": reason
    }

# Configuration
config = {
    "symbol": "BTCUSDT",
    "interval": "5m",
    "description": "Advanced indicators demo strategy using ADX, Parabolic SAR, Keltner Channels, OBV, Aroon, and Ichimoku"
}
