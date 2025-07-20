# Multi-Indicator Strategy
# Demonstrates usage of multiple new indicators

# Strategy settings configuration
def settings():
    return {
        "interval": "5m",  # Default kline interval - can be overridden in config
        "rsi_period": 14,
        "rsi_oversold": 30,
        "rsi_overbought": 70,
        "atr_period": 14,
        "cci_period": 20,
        "cci_oversold": -100,
        "cci_overbought": 100,
        "williams_r_period": 14,
        "williams_r_oversold": -80,
        "williams_r_overbought": -20,
        "mfi_period": 14,
        "mfi_oversold": 20,
        "mfi_overbought": 80,
        "roc_period": 12,
        "volatility_threshold": 2.0,  # ATR threshold for volatility filter
        "position_size": 0.01
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
rsi_period = params["rsi_period"]  # Default values, will be overridden by config
rsi_oversold = params["rsi_oversold"]
rsi_overbought = params["rsi_overbought"]
atr_period = params["atr_period"]
cci_period = params["cci_period"]
cci_oversold = params["cci_oversold"]
cci_overbought = params["cci_overbought"]
williams_r_period = params["williams_r_period"]
williams_r_oversold = params["williams_r_oversold"]
williams_r_overbought = params["williams_r_overbought"]
mfi_period = params["mfi_period"]
mfi_oversold = params["mfi_oversold"]
mfi_overbought = params["mfi_overbought"]
roc_period = params["roc_period"]
volatility_threshold = params["volatility_threshold"]
position_size = params["position_size"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "rsi_period": get_config("rsi_period", params["rsi_period"]),
        "rsi_oversold": get_config("rsi_oversold", params["rsi_oversold"]),
        "rsi_overbought": get_config("rsi_overbought", params["rsi_overbought"]),
        "atr_period": get_config("atr_period", params["atr_period"]),
        "cci_period": get_config("cci_period", params["cci_period"]),
        "cci_oversold": get_config("cci_oversold", params["cci_oversold"]),
        "cci_overbought": get_config("cci_overbought", params["cci_overbought"]),
        "williams_r_period": get_config("williams_r_period", params["williams_r_period"]),
        "williams_r_oversold": get_config("williams_r_oversold", params["williams_r_oversold"]),
        "williams_r_overbought": get_config("williams_r_overbought", params["williams_r_overbought"]),
        "mfi_period": get_config("mfi_period", params["mfi_period"]),
        "mfi_oversold": get_config("mfi_oversold", params["mfi_oversold"]),
        "mfi_overbought": get_config("mfi_overbought", params["mfi_overbought"]),
        "roc_period": get_config("roc_period", params["roc_period"]),
        "volatility_threshold": get_config("volatility_threshold", params["volatility_threshold"]),
        "position_size": get_config("position_size", params["position_size"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("klines", [])
        set_state("last_signal", "hold")
        set_state("indicators", {})

def on_kline(kline):
    """Called when a new kline is received"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    
    # Get current state
    klines = get_state("klines", [])
    
    # Add new kline to our buffer
    klines.append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only the klines we need
    max_needed = max(cfg["rsi_period"], cfg["atr_period"], cfg["cci_period"], cfg["williams_r_period"], cfg["mfi_period"], cfg["roc_period"]) + 20
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Extract price and volume data
    closes = [k["close"] for k in klines]
    highs = [k["high"] for k in klines]
    lows = [k["low"] for k in klines]
    volumes = [k["volume"] for k in klines]
    
    # Calculate indicators if we have enough data
    if len(closes) >= max_needed - 10:
        calculate_indicators(closes, highs, lows, volumes, cfg)
    
    # Check for trading signals
    return check_signals(closes[-1] if closes else 0, cfg)

def calculate_indicators(closes, highs, lows, volumes, cfg):
    """Calculate all technical indicators"""
    indicators = get_state("indicators", {})
    
    # Calculate RSI
    if len(closes) >= cfg["rsi_period"]:
        indicators["rsi"] = rsi(closes, cfg["rsi_period"])
    
    # Calculate ATR for volatility filter
    if len(closes) >= cfg["atr_period"]:
        indicators["atr"] = atr(highs, lows, closes, cfg["atr_period"])
    
    # Calculate CCI
    if len(closes) >= cfg["cci_period"]:
        indicators["cci"] = cci(highs, lows, closes, cfg["cci_period"])
    
    # Calculate Williams %R
    if len(closes) >= cfg["williams_r_period"]:
        indicators["williams_r"] = williams_r(highs, lows, closes, cfg["williams_r_period"])
    
    # Calculate MFI (Money Flow Index)
    if len(volumes) >= cfg["mfi_period"] and len(closes) >= cfg["mfi_period"]:
        indicators["mfi"] = mfi(highs, lows, closes, volumes, cfg["mfi_period"])
    
    # Calculate Rate of Change
    if len(closes) >= cfg["roc_period"]:
        indicators["roc"] = roc(closes, cfg["roc_period"])
    
    # Calculate VWAP
    if len(volumes) > 0:
        indicators["vwap"] = vwap(highs, lows, closes, volumes)
    
    # Calculate Standard Deviation for volatility analysis
    if len(closes) >= 20:
        indicators["stddev"] = stddev(closes, 20)
    
    # Update state
    set_state("indicators", indicators)

def check_signals(current_price, cfg):
    """Check for trading signals using multiple indicators"""
    indicators = get_state("indicators", {})
    
    if not indicators:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": current_price,
            "type": "market",
            "reason": "Insufficient data for indicators"
        }
    
    # Get current indicator values
    current_rsi = get_last_value("rsi", indicators)
    current_atr = get_last_value("atr", indicators)
    current_cci = get_last_value("cci", indicators)
    current_williams_r = get_last_value("williams_r", indicators)
    current_mfi = get_last_value("mfi", indicators)
    current_roc = get_last_value("roc", indicators)
    current_vwap = get_last_value("vwap", indicators)
    current_stddev = get_last_value("stddev", indicators)
    
    # Volatility filter - only trade when volatility is reasonable
    if current_atr and current_stddev:
        volatility_ratio = current_atr / (current_price / 100)  # ATR as percentage of price
        if volatility_ratio > cfg["volatility_threshold"]:
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": current_price,
                "type": "market",
                "reason": "High volatility - ATR ratio: " + str(round(volatility_ratio, 3))
            }
    
    # Multi-indicator bullish signals
    bullish_signals = 0
    bearish_signals = 0
    signal_reasons = []
    
    # RSI analysis
    if current_rsi is not None:
        if current_rsi < cfg["rsi_oversold"]:
            bullish_signals += 1
            signal_reasons.append("RSI oversold (" + str(round(current_rsi, 1)) + ")")
        elif current_rsi > cfg["rsi_overbought"]:
            bearish_signals += 1
            signal_reasons.append("RSI overbought (" + str(round(current_rsi, 1)) + ")")
    
    # CCI analysis
    if current_cci is not None:
        if current_cci < cfg["cci_oversold"]:
            bullish_signals += 1
            signal_reasons.append("CCI oversold (" + str(round(current_cci, 1)) + ")")
        elif current_cci > cfg["cci_overbought"]:
            bearish_signals += 1
            signal_reasons.append("CCI overbought (" + str(round(current_cci, 1)) + ")")
    
    # Williams %R analysis
    if current_williams_r is not None:
        if current_williams_r < cfg["williams_r_oversold"]:
            bullish_signals += 1
            signal_reasons.append("Williams %R oversold (" + str(round(current_williams_r, 1)) + ")")
        elif current_williams_r > cfg["williams_r_overbought"]:
            bearish_signals += 1
            signal_reasons.append("Williams %R overbought (" + str(round(current_williams_r, 1)) + ")")
    
    # MFI analysis (volume-weighted momentum)
    if current_mfi is not None:
        if current_mfi < cfg["mfi_oversold"]:
            bullish_signals += 1
            signal_reasons.append("MFI oversold (" + str(round(current_mfi, 1)) + ")")
        elif current_mfi > cfg["mfi_overbought"]:
            bearish_signals += 1
            signal_reasons.append("MFI overbought (" + str(round(current_mfi, 1)) + ")")
    
    # Rate of Change analysis (momentum)
    if current_roc is not None:
        if current_roc < -5:  # Strong negative momentum
            bearish_signals += 1
            signal_reasons.append("Strong negative momentum ROC (" + str(round(current_roc, 1)) + "%)")
        elif current_roc > 5:  # Strong positive momentum
            bullish_signals += 1
            signal_reasons.append("Strong positive momentum ROC (" + str(round(current_roc, 1)) + "%)")
    
    # VWAP analysis (institutional interest)
    if current_vwap is not None:
        if current_price > current_vwap * 1.002:  # Price above VWAP
            bullish_signals += 0.5
            signal_reasons.append("Price above VWAP")
        elif current_price < current_vwap * 0.998:  # Price below VWAP
            bearish_signals += 0.5
            signal_reasons.append("Price below VWAP")
    
    # Decision logic - require multiple confirmations
    signal_threshold = 2  # Require at least 2 indicators to agree
    
    if bullish_signals >= signal_threshold and bullish_signals > bearish_signals:
        reason = "Multiple bullish signals: " + ", ".join(signal_reasons)
        log("BUY signal: " + reason)
        set_state("last_signal", "buy")
        return {
            "action": "buy",
            "quantity": cfg["position_size"],
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    elif bearish_signals >= signal_threshold and bearish_signals > bullish_signals:
        reason = "Multiple bearish signals: " + ", ".join(signal_reasons)
        log("SELL signal: " + reason)
        set_state("last_signal", "sell")
        return {
            "action": "sell",
            "quantity": cfg["position_size"],
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    # No clear signal
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No consensus signal (Bull: " + str(bullish_signals) + ", Bear: " + str(bearish_signals) + ")"
    }

def get_last_value(indicator_name, indicators):
    """Get the last value of an indicator, handling None/empty cases"""
    if indicator_name not in indicators:
        return None
    
    indicator_values = indicators[indicator_name]
    if not indicator_values or len(indicator_values) == 0:
        return None
    
    last_value = indicator_values[-1]
    # Handle NaN values (they would be None in Starlark)
    if last_value is None:
        return None
    
    return last_value

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    # Use orderbook for execution timing
    if len(orderbook.bids) > 0 and len(orderbook.asks) > 0:
        spread = orderbook.asks[0].price - orderbook.bids[0].price
        mid_price = (orderbook.bids[0].price + orderbook.asks[0].price) / 2
        spread_percent = (spread / mid_price) * 100
        
        # Only trade when spread is reasonable (less than 0.2%)
        if spread_percent > 0.2:
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": 0.0,
                "type": "market",
                "reason": "Spread too wide for execution: " + str(round(spread_percent, 3)) + "%"
            }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "Orderbook conditions acceptable"
    }

def on_ticker(ticker):
    """Called when ticker data is received"""
    # Use volume for signal confirmation
    if ticker.volume > 0:
        # Log volume information for analysis
        log("Volume analysis - Current: " + str(ticker.volume))
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": ticker.price,
        "type": "market",
        "reason": "Ticker data processed"
    }