# Advanced Multi-Indicator Strategy using new MarketMaestro indicators
# This strategy demonstrates the use of newly added technical indicators

def settings():
    return {
        "interval": "15m",
        "rvi_period": 14,
        "cmf_period": 20,
        "lr_period": 14,
        "bb_period": 20,
        "bb_multiplier": 2.0,
        "position_size": 0.01,
        "min_confirmations": 2
    }

def get_config_values():
    """Get configuration values with fallbacks"""
    params = settings()
    return {
        "rvi_period": get_config("rvi_period", params["rvi_period"]),
        "cmf_period": get_config("cmf_period", params["cmf_period"]),
        "lr_period": get_config("lr_period", params["lr_period"]),
        "bb_period": get_config("bb_period", params["bb_period"]),
        "bb_multiplier": get_config("bb_multiplier", params["bb_multiplier"]),
        "position_size": get_config("position_size", params["position_size"]),
        "min_confirmations": get_config("min_confirmations", params["min_confirmations"])
    }

def init_state():
    """Initialize thread-safe state"""
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("position", 0)
        set_state("last_signal", "hold")
        set_state("entry_price", 0.0)

def on_kline(kline):
    """Advanced multi-indicator strategy using new indicators"""
    init_state()
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    position = get_state("position", 0)
    
    # Add new kline
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep required history
    max_needed = max(cfg["rvi_period"], cfg["cmf_period"], cfg["lr_period"], cfg["bb_period"]) + 10
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    set_state("klines", klines)
    
    # Check data sufficiency
    if len(klines) < max_needed - 10:
        return {"action": "hold", "reason": "Insufficient data"}
    
    # Extract price arrays
    open_prices = [k["open"] for k in klines]
    high_prices = [k["high"] for k in klines]
    low_prices = [k["low"] for k in klines]
    close_prices = [k["close"] for k in klines]
    volume_data = [k["volume"] for k in klines]
    
    current_price = kline.close
    
    # Calculate new technical indicators
    try:
        # Relative Vigor Index - momentum
        rvi_data = rvi(open_prices, high_prices, low_prices, close_prices, cfg["rvi_period"])
        rvi_line = rvi_data["rvi"][-1] if rvi_data["rvi"] else 0
        rvi_signal = rvi_data["signal"][-1] if rvi_data["signal"] else 0
        
        # Chaikin Money Flow - volume-based
        cmf_line = chaikin_money_flow(high_prices, low_prices, close_prices, volume_data, cfg["cmf_period"])
        current_cmf = cmf_line[-1] if cmf_line else 0
        
        # Linear Regression Slope - trend direction
        lr_slope = linear_regression_slope(close_prices, cfg["lr_period"])
        current_slope = lr_slope[-1] if lr_slope else 0
        
        # Bollinger %B - position within bands
        percent_b = bollinger_percent_b(close_prices, cfg["bb_period"], cfg["bb_multiplier"])
        current_b = percent_b[-1] if percent_b else 0.5
        
        # Bollinger Band Width - volatility
        bb_width = bollinger_band_width(close_prices, cfg["bb_period"], cfg["bb_multiplier"])
        current_width = bb_width[-1] if bb_width else 0
        avg_width = sum(bb_width[-10:]) / 10 if len(bb_width) >= 10 else current_width
        
    except Exception as e:
        log(f"Error calculating indicators: {e}")
        return {"action": "hold", "reason": "Indicator calculation error"}
    
    # Signal analysis
    confirmations = 0
    signal_reasons = []
    
    # RVI momentum confirmation
    if rvi_line > rvi_signal and rvi_line > 0:
        confirmations += 1
        signal_reasons.append(f"RVI bullish: {round(rvi_line, 4)}")
    elif rvi_line < rvi_signal and rvi_line < 0:
        confirmations -= 1
        signal_reasons.append(f"RVI bearish: {round(rvi_line, 4)}")
    
    # Chaikin Money Flow confirmation
    if current_cmf > 0.1:
        confirmations += 1
        signal_reasons.append(f"CMF bullish: {round(current_cmf, 4)}")
    elif current_cmf < -0.1:
        confirmations -= 1
        signal_reasons.append(f"CMF bearish: {round(current_cmf, 4)}")
    
    # Linear Regression trend confirmation
    if current_slope > 0.01:
        confirmations += 1
        signal_reasons.append(f"Trend up: {round(current_slope, 4)}")
    elif current_slope < -0.01:
        confirmations -= 1
        signal_reasons.append(f"Trend down: {round(current_slope, 4)}")
    
    # Bollinger Band position analysis
    bb_signal = ""
    if current_b < 0.2:  # Near lower band
        confirmations += 0.5
        bb_signal = f"Near lower BB: {round(current_b, 2)}"
    elif current_b > 0.8:  # Near upper band
        confirmations -= 0.5
        bb_signal = f"Near upper BB: {round(current_b, 2)}"
    
    # Volatility filter (avoid trading in low volatility)
    is_low_volatility = current_width < avg_width * 0.7
    if is_low_volatility:
        return {
            "action": "hold",
            "reason": f"Low volatility filter: width={round(current_width, 4)}"
        }
    
    # Entry logic
    if position == 0:
        if confirmations >= cfg["min_confirmations"]:
            set_state("position", 1)
            set_state("entry_price", current_price)
            set_state("last_signal", "buy")
            
            reason = f"BUY - Confirmations: {confirmations}. " + "; ".join(signal_reasons)
            if bb_signal:
                reason += f"; {bb_signal}"
            
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": reason
            }
        
        elif confirmations <= -cfg["min_confirmations"]:
            set_state("position", -1)
            set_state("entry_price", current_price)
            set_state("last_signal", "sell")
            
            reason = f"SELL - Confirmations: {confirmations}. " + "; ".join(signal_reasons)
            if bb_signal:
                reason += f"; {bb_signal}"
            
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": reason
            }
    
    # Exit logic
    elif position != 0:
        entry_price = get_state("entry_price", current_price)
        pnl_percent = ((current_price - entry_price) / entry_price * 100) * position
        
        # Exit on opposite signals or significant profit/loss
        should_exit = False
        exit_reason = ""
        
        if position == 1 and confirmations <= -1:
            should_exit = True
            exit_reason = "Bearish signals"
        elif position == -1 and confirmations >= 1:
            should_exit = True
            exit_reason = "Bullish signals"
        elif abs(pnl_percent) > 3:  # 3% profit/loss
            should_exit = True
            exit_reason = f"P&L target: {round(pnl_percent, 2)}%"
        
        if should_exit:
            set_state("position", 0)
            set_state("entry_price", 0.0)
            set_state("last_signal", "exit")
            
            action = "sell" if position == 1 else "buy"
            return {
                "action": action,
                "quantity": cfg["position_size"],
                "type": "market",
                "reason": f"EXIT - {exit_reason}. P&L: {round(pnl_percent, 2)}%"
            }
    
    # Hold position
    status_parts = []
    if signal_reasons:
        status_parts.append(f"Signals: {'; '.join(signal_reasons[:2])}")
    status_parts.append(f"Confirmations: {confirmations}")
    if current_b is not None:
        status_parts.append(f"BB %B: {round(current_b, 2)}")
    
    return {
        "action": "hold",
        "reason": "HOLD - " + "; ".join(status_parts)
    }

def on_start():
    """Initialize strategy when it starts"""
    log("🚀 Advanced Multi-Indicator Strategy starting")
    log("📊 Using RVI, CMF, Linear Regression, and Bollinger Band derivatives")
    init_state()
    log("✅ Strategy initialization complete")

def on_stop():
    """Clean up when strategy stops"""
    log("🛑 Advanced Multi-Indicator Strategy stopping")
    position = get_state("position", 0)
    if position != 0:
        log(f"⚠️ Stopping with open position: {position}")
    log("✅ Strategy stopped cleanly")