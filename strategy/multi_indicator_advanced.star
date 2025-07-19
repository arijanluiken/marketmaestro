# Multi-Indicator Advanced Strategy
# This strategy demonstrates many of the new technical indicators

def settings():
    return {
        "interval": "15m",
        "symbol": "BTCUSDT",
        "description": "Advanced multi-indicator strategy using trend, momentum, and volatility analysis"
    }

def on_kline(kline):
    # Get sufficient price data
    high = [float(k.high) for k in get_historical_klines(50)]
    low = [float(k.low) for k in get_historical_klines(50)]
    close = [float(k.close) for k in get_historical_klines(50)]
    open_prices = [float(k.open) for k in get_historical_klines(50)]
    volume = [float(k.volume) for k in get_historical_klines(50)]
    
    if len(close) < 50:
        return {"action": "hold", "reason": "insufficient data"}
    
    current_price = float(kline.close)
    
    # === TREND ANALYSIS ===
    
    # Supertrend for primary trend
    supertrend_result = supertrend(high, low, close, 10, 3.0)
    supertrend_line = supertrend_result["supertrend"][-1] if len(supertrend_result["supertrend"]) > 0 else 0
    is_uptrend = supertrend_result["trend"][-1] if len(supertrend_result["trend"]) > 0 else False
    
    # Williams Alligator for trend confirmation
    alligator = williams_alligator(close)
    jaw = alligator["jaw"][-1] if len(alligator["jaw"]) > 0 else 0
    teeth = alligator["teeth"][-1] if len(alligator["teeth"]) > 0 else 0
    lips = alligator["lips"][-1] if len(alligator["lips"]) > 0 else 0
    
    # Vortex for trend strength
    vortex_result = vortex(high, low, close, 14)
    vi_plus = vortex_result["vi_plus"][-1] if len(vortex_result["vi_plus"]) > 0 else 0
    vi_minus = vortex_result["vi_minus"][-1] if len(vortex_result["vi_minus"]) > 0 else 0
    
    # === MOMENTUM ANALYSIS ===
    
    # Stochastic RSI for overbought/oversold
    stoch_rsi = stochastic_rsi(close, 14, 14, 3, 3)
    stoch_k = stoch_rsi["k"][-1] if len(stoch_rsi["k"]) > 0 else 50
    stoch_d = stoch_rsi["d"][-1] if len(stoch_rsi["d"]) > 0 else 50
    
    # Awesome Oscillator for momentum
    ao = awesome_oscillator(high, low)
    current_ao = ao[-1] if len(ao) > 0 else 0
    previous_ao = ao[-2] if len(ao) > 1 else 0
    
    # Accelerator Oscillator for momentum acceleration
    ac = accelerator_oscillator(high, low, close)
    current_ac = ac[-1] if len(ac) > 0 else 0
    
    # === CANDLESTICK ANALYSIS ===
    
    # Heikin Ashi for cleaner trend signals
    ha = heikin_ashi(open_prices, high, low, close)
    ha_close = ha["close"][-1] if len(ha["close"]) > 0 else 0
    ha_open = ha["open"][-1] if len(ha["open"]) > 0 else 0
    ha_bullish = ha_close > ha_open
    
    # === ADVANCED INDICATORS ===
    
    # TSI for trend strength
    tsi_values = tsi(close, 25, 13)
    current_tsi = tsi_values[-1] if len(tsi_values) > 0 else 0
    
    # Elder Ray for bull/bear power
    elder = elder_ray(high, low, close, 13)
    bull_power = elder["bull_power"][-1] if len(elder["bull_power"]) > 0 else 0
    bear_power = elder["bear_power"][-1] if len(elder["bear_power"]) > 0 else 0
    
    # === SIGNAL GENERATION ===
    
    # Strong bullish signals
    strong_bullish = (
        is_uptrend and 
        current_price > supertrend_line and
        lips > teeth > jaw and  # Alligator trending up
        vi_plus > vi_minus and  # Vortex bullish
        stoch_k < 80 and  # Not overbought
        current_ao > 0 and  # Positive momentum
        current_ac > 0 and  # Accelerating momentum
        ha_bullish and  # Heikin Ashi bullish
        current_tsi > -25 and  # TSI not oversold
        bull_power > 0  # Bulls in control
    )
    
    # Strong bearish signals
    strong_bearish = (
        not is_uptrend and
        current_price < supertrend_line and
        lips < teeth < jaw and  # Alligator trending down
        vi_minus > vi_plus and  # Vortex bearish
        stoch_k > 20 and  # Not oversold
        current_ao < 0 and  # Negative momentum
        current_ac < 0 and  # Accelerating downward
        not ha_bullish and  # Heikin Ashi bearish
        current_tsi < 25 and  # TSI not overbought
        bear_power < 0  # Bears in control
    )
    
    # Log all indicator values for analysis
    log(f"Price: {current_price:.2f}")
    log(f"Supertrend: {supertrend_line:.2f}, Uptrend: {is_uptrend}")
    log(f"Alligator - Jaw: {jaw:.2f}, Teeth: {teeth:.2f}, Lips: {lips:.2f}")
    log(f"Vortex - VI+: {vi_plus:.3f}, VI-: {vi_minus:.3f}")
    log(f"Stoch RSI K: {stoch_k:.1f}, D: {stoch_d:.1f}")
    log(f"AO: {current_ao:.4f}, AC: {current_ac:.4f}")
    log(f"Heikin Ashi Bullish: {ha_bullish}")
    log(f"TSI: {current_tsi:.2f}")
    log(f"Elder Ray - Bull: {bull_power:.4f}, Bear: {bear_power:.4f}")
    
    # Execute trades based on signals
    if strong_bullish:
        log(f"STRONG BUY signal at {current_price}")
        signal("buy", 0.02)  # Larger position for strong signals
        
    elif strong_bearish:
        log(f"STRONG SELL signal at {current_price}")
        signal("sell", 0.02)
        
    # Weaker signals for smaller positions
    elif (is_uptrend and current_price > supertrend_line and 
          stoch_k < 70 and current_ao > previous_ao):
        log(f"Weak BUY signal at {current_price}")
        signal("buy", 0.01)
        
    elif (not is_uptrend and current_price < supertrend_line and 
          stoch_k > 30 and current_ao < previous_ao):
        log(f"Weak SELL signal at {current_price}")
        signal("sell", 0.01)
    
    else:
        return {"action": "hold", "reason": "mixed signals"}

def on_orderbook(orderbook):
    # Could add spread analysis here
    pass

def on_ticker(ticker):
    # Could add volume analysis here
    pass
