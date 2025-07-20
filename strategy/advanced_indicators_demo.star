# Advanced Indicators Demo Strategy
# Demonstrates the new technical indicators available in the strategy engine
# Updated to use callback-provided data instead of global variables

state = {
    "klines": []
}

def settings():
    return {
        "interval": "5m",
        "adx_period": 14,
        "keltner_period": 20,
        "aroon_period": 14,
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
    
    # Keep only what we need for calculations
    max_needed = 100
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Check if we have enough data
    if len(state["klines"]) < 50:
        return {"action": "hold", "reason": "not enough data points"}
    
    # Extract price arrays from internal buffer
    highs = [k["high"] for k in state["klines"]]
    lows = [k["low"] for k in state["klines"]]
    closes = [k["close"] for k in state["klines"]]
    volumes = [k["volume"] for k in state["klines"]]
    
    # Calculate various indicators
    
    # 1. ADX for trend strength
    adx_result = adx(highs, lows, closes, config.get("adx_period", 14))
    adx_line = adx_result["adx"]
    plus_di = adx_result["plus_di"]
    minus_di = adx_result["minus_di"]
    
    # 2. Parabolic SAR for trend direction
    psar = parabolic_sar(highs, lows, step=0.02, max_step=0.2)
    
    # 3. Keltner Channels for volatility
    keltner_result = keltner_channel(highs, lows, closes, config.get("keltner_period", 20), multiplier=2.0)
    keltner_upper = keltner_result["upper"]
    keltner_middle = keltner_result["middle"]
    keltner_lower = keltner_result["lower"]
    
    # 4. On-Balance Volume for volume analysis
    obv_line = obv(closes, volumes)
    
    # 5. Aroon for momentum
    aroon_result = aroon(highs, lows, config.get("aroon_period", 14))
    aroon_up = aroon_result["aroon_up"]
    aroon_down = aroon_result["aroon_down"]
    
    # 6. Ichimoku Cloud components
    ichimoku_result = ichimoku(highs, lows, closes, 
                              conversion_period=9, 
                              base_period=26, 
                              span_b_period=52, 
                              displacement=26)
    tenkan_sen = ichimoku_result["tenkan_sen"]
    kijun_sen = ichimoku_result["kijun_sen"]
    
    # Get current values (last data point)
    current_price = closes[-1]
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
    
    position_size = config.get("position_size", 0.01)
    
    # Decision making
    if bullish_conditions >= 6:
        action = "buy"
        reason = "Strong bullish signals: " + str(bullish_conditions) + "/8 conditions met. ADX: " + str(round(current_adx, 2)) + ", Price vs PSAR: " + ("above" if current_price > current_psar else "below") + ", OBV trend: " + obv_trend
    elif bearish_conditions >= 6:
        action = "sell"
        reason = "Strong bearish signals: " + str(bearish_conditions) + "/8 conditions met. ADX: " + str(round(current_adx, 2)) + ", Price vs PSAR: " + ("above" if current_price > current_psar else "below") + ", OBV trend: " + obv_trend
    else:
        action = "hold"
        reason = "Mixed signals - Bullish: " + str(bullish_conditions) + ", Bearish: " + str(bearish_conditions) + ". ADX: " + str(round(current_adx, 2)) + " (trend strength)"
    
    return {
        "action": action,
        "quantity": position_size,
        "price": current_price,
        "type": "market",
        "reason": reason
    }

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {"action": "hold", "reason": "No ticker signal"}
