# Advanced Technical Indicators Test Strategy
# This strategy demonstrates the new technical indicators and lifecycle callbacks

def settings():
    return {
        "interval": "5m",
        "symbol": "BTCUSDT",
        "description": "Test strategy using advanced technical indicators with lifecycle callbacks"
    }

def on_start():
    """
    Called when the strategy starts. Use this for initialization.
    """
    print("=== Strategy Starting ===")
    print("Initializing advanced indicators strategy")
    print("This strategy will test TSI, Donchian Channels, Advanced CCI, Elder Ray, and more")
    print("========================")

def on_stop():
    """
    Called when the strategy stops. Use this for cleanup.
    """
    print("=== Strategy Stopping ===")
    print("Cleaning up advanced indicators strategy")
    print("Final cleanup complete")
    print("=========================")

def on_kline(kline):
    # Use klines buffer from context
    if len(klines) < 20:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market",
            "reason": "Insufficient historical data"
        }
    
    # Extract price data from klines buffer
    high = [float(k["high"]) for k in klines[-20:]]
    low = [float(k["low"]) for k in klines[-20:]]
    close = [float(k["close"]) for k in klines[-20:]]
    volume = [float(k["volume"]) for k in klines[-20:]]
    
    # Test True Strength Index (TSI)
    tsi_values = tsi(close, 25, 13)
    current_tsi = tsi_values[-1] if len(tsi_values) > 0 else 0
    log("TSI: " + str(current_tsi))
    
    # Test Donchian Channels
    donchian_result = donchian(high, low, 20)
    upper_channel = donchian_result["upper"][-1] if len(donchian_result["upper"]) > 0 else 0
    lower_channel = donchian_result["lower"][-1] if len(donchian_result["lower"]) > 0 else 0
    log("Donchian Upper: " + str(upper_channel) + ", Lower: " + str(lower_channel))
    
    # Test Advanced CCI with smoothing
    advanced_cci_values = advanced_cci(high, low, close, 14, 3)
    current_cci = advanced_cci_values[-1] if len(advanced_cci_values) > 0 else 0
    log("Advanced CCI: " + str(current_cci))
    
    # Test Elder Ray Index
    elder_result = elder_ray(high, low, close, 13)
    bull_power = elder_result["bull_power"][-1] if len(elder_result["bull_power"]) > 0 else 0
    bear_power = elder_result["bear_power"][-1] if len(elder_result["bear_power"]) > 0 else 0
    log("Elder Ray - Bull Power: " + str(bull_power) + ", Bear Power: " + str(bear_power))
    
    # Test Detrended Price Oscillator
    detrended_values = detrended(close, 14)
    current_detrended = detrended_values[-1] if len(detrended_values) > 0 else 0
    log("Detrended Price Oscillator: " + str(current_detrended))
    
    # Test Kaufman Adaptive Moving Average (KAMA)
    kama_values = kama(close, 10, 2, 30)
    current_kama = kama_values[-1] if len(kama_values) > 0 else 0
    log("KAMA: " + str(current_kama))
    
    # Test Chaikin Oscillator
    chaikin_values = chaikin_oscillator(high, low, close, volume, 3, 10)
    current_chaikin = chaikin_values[-1] if len(chaikin_values) > 0 else 0
    log("Chaikin Oscillator: " + str(current_chaikin))
    
    # Test Ultimate Oscillator
    ultimate_values = ultimate_oscillator(high, low, close, 7, 14, 28)
    current_ultimate = ultimate_values[-1] if len(ultimate_values) > 0 else 0
    log("Ultimate Oscillator: " + str(current_ultimate))
    
    # Simple trading logic based on multiple indicators
    current_price = float(kline.close)
    
    # Buy signal: TSI crossing above -25, price above lower Donchian, and bull power positive
    if current_tsi > -25 and current_price > lower_channel and bull_power > 0:
        log("BUY signal at " + str(current_price) + " - TSI: " + str(current_tsi) + ", Bull Power: " + str(bull_power))
        return {
            "action": "buy",
            "quantity": 0.01,
            "price": current_price,
            "type": "market",
            "reason": "Advanced indicators bullish: TSI=" + str(round(current_tsi, 2)) + " BullPower=" + str(round(bull_power, 4))
        }
    
    # Sell signal: TSI crossing below 25, price below upper Donchian, and bear power negative
    elif current_tsi < 25 and current_price < upper_channel and bear_power < 0:
        log("SELL signal at " + str(current_price) + " - TSI: " + str(current_tsi) + ", Bear Power: " + str(bear_power))
        return {
            "action": "sell",
            "quantity": 0.01,
            "price": current_price,
            "type": "market",
            "reason": "Advanced indicators bearish: TSI=" + str(round(current_tsi, 2)) + " BearPower=" + str(round(bear_power, 4))
        }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No signal from advanced indicators"
    }

def on_orderbook(orderbook):
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No orderbook signal"
    }

def on_ticker(ticker):
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No ticker signal"
    }
