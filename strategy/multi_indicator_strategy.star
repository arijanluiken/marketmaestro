# Multi-Indicator Strategy
# Demonstrates usage of multiple new indicators

# Strategy settings configuration
def settings():
    return {
        "interval": "5m",  # Kline interval for this strategy
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

# Strategy state
state = {
    "klines": [],
    "last_signal": "hold",
    "indicators": {}
}

# Initialize strategy parameters
params = settings()
rsi_period = config.get("rsi_period", params["rsi_period"])
rsi_oversold = config.get("rsi_oversold", params["rsi_oversold"])
rsi_overbought = config.get("rsi_overbought", params["rsi_overbought"])
atr_period = config.get("atr_period", params["atr_period"])
cci_period = config.get("cci_period", params["cci_period"])
cci_oversold = config.get("cci_oversold", params["cci_oversold"])
cci_overbought = config.get("cci_overbought", params["cci_overbought"])
williams_r_period = config.get("williams_r_period", params["williams_r_period"])
williams_r_oversold = config.get("williams_r_oversold", params["williams_r_oversold"])
williams_r_overbought = config.get("williams_r_overbought", params["williams_r_overbought"])
mfi_period = config.get("mfi_period", params["mfi_period"])
mfi_oversold = config.get("mfi_oversold", params["mfi_oversold"])
mfi_overbought = config.get("mfi_overbought", params["mfi_overbought"])
roc_period = config.get("roc_period", params["roc_period"])
volatility_threshold = config.get("volatility_threshold", params["volatility_threshold"])
position_size = config.get("position_size", params["position_size"])

def on_kline(kline):
    """Called when a new kline is received"""
    # Add new kline to our buffer
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep only the klines we need
    max_needed = max(rsi_period, atr_period, cci_period, williams_r_period, mfi_period, roc_period) + 20
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Extract price and volume data
    closes = [k["close"] for k in state["klines"]]
    highs = [k["high"] for k in state["klines"]]
    lows = [k["low"] for k in state["klines"]]
    volumes = [k["volume"] for k in state["klines"]]
    
    # Calculate indicators if we have enough data
    if len(closes) >= max_needed - 10:
        calculate_indicators(closes, highs, lows, volumes)
    
    # Check for trading signals
    return check_signals(closes[-1] if closes else 0)

def calculate_indicators(closes, highs, lows, volumes):
    """Calculate all technical indicators"""
    # Calculate RSI
    if len(closes) >= rsi_period:
        state["indicators"]["rsi"] = rsi(closes, rsi_period)
    
    # Calculate ATR for volatility filter
    if len(closes) >= atr_period:
        state["indicators"]["atr"] = atr(highs, lows, closes, atr_period)
    
    # Calculate CCI
    if len(closes) >= cci_period:
        state["indicators"]["cci"] = cci(highs, lows, closes, cci_period)
    
    # Calculate Williams %R
    if len(closes) >= williams_r_period:
        state["indicators"]["williams_r"] = williams_r(highs, lows, closes, williams_r_period)
    
    # Calculate MFI (Money Flow Index)
    if len(volumes) >= mfi_period and len(closes) >= mfi_period:
        state["indicators"]["mfi"] = mfi(highs, lows, closes, volumes, mfi_period)
    
    # Calculate Rate of Change
    if len(closes) >= roc_period:
        state["indicators"]["roc"] = roc(closes, roc_period)
    
    # Calculate VWAP
    if len(volumes) > 0:
        state["indicators"]["vwap"] = vwap(highs, lows, closes, volumes)
    
    # Calculate Standard Deviation for volatility analysis
    if len(closes) >= 20:
        state["indicators"]["stddev"] = stddev(closes, 20)

def check_signals(current_price):
    """Check for trading signals using multiple indicators"""
    if not state["indicators"]:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": current_price,
            "type": "market",
            "reason": "Insufficient data for indicators"
        }
    
    # Get current indicator values
    current_rsi = get_last_value("rsi")
    current_atr = get_last_value("atr")
    current_cci = get_last_value("cci")
    current_williams_r = get_last_value("williams_r")
    current_mfi = get_last_value("mfi")
    current_roc = get_last_value("roc")
    current_vwap = get_last_value("vwap")
    current_stddev = get_last_value("stddev")
    
    # Volatility filter - only trade when volatility is reasonable
    if current_atr and current_stddev:
        volatility_ratio = current_atr / (current_price / 100)  # ATR as percentage of price
        if volatility_ratio > volatility_threshold:
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": current_price,
                "type": "market",
                "reason": "High volatility - ATR ratio: " + str(round(volatility_ratio, 2))
            }
    
    # Multi-indicator bullish signals
    bullish_signals = 0
    bearish_signals = 0
    signal_reasons = []
    
    # RSI analysis
    if current_rsi is not None:
        if current_rsi < rsi_oversold:
            bullish_signals += 1
            signal_reasons.append("RSI oversold (" + str(round(current_rsi, 1)) + ")")
        elif current_rsi > rsi_overbought:
            bearish_signals += 1
            signal_reasons.append("RSI overbought (" + str(round(current_rsi, 1)) + ")")
    
    # CCI analysis
    if current_cci is not None:
        if current_cci < cci_oversold:
            bullish_signals += 1
            signal_reasons.append("CCI oversold (" + str(round(current_cci, 1)) + ")")
        elif current_cci > cci_overbought:
            bearish_signals += 1
            signal_reasons.append("CCI overbought (" + str(round(current_cci, 1)) + ")")
    
    # Williams %R analysis
    if current_williams_r is not None:
        if current_williams_r < williams_r_oversold:
            bullish_signals += 1
            signal_reasons.append("Williams %R oversold (" + str(round(current_williams_r, 1)) + ")")
        elif current_williams_r > williams_r_overbought:
            bearish_signals += 1
            signal_reasons.append("Williams %R overbought (" + str(round(current_williams_r, 1)) + ")")
    
    # MFI analysis (volume-weighted momentum)
    if current_mfi is not None:
        if current_mfi < mfi_oversold:
            bullish_signals += 1
            signal_reasons.append("MFI oversold (" + str(round(current_mfi, 1)) + ")")
        elif current_mfi > mfi_overbought:
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
        state["last_signal"] = "buy"
        return {
            "action": "buy",
            "quantity": position_size,
            "price": current_price,
            "type": "market",
            "reason": reason
        }
    
    elif bearish_signals >= signal_threshold and bearish_signals > bullish_signals:
        reason = "Multiple bearish signals: " + ", ".join(signal_reasons)
        log("SELL signal: " + reason)
        state["last_signal"] = "sell"
        return {
            "action": "sell",
            "quantity": position_size,
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

def get_last_value(indicator_name):
    """Get the last value of an indicator, handling None/empty cases"""
    if indicator_name not in state["indicators"]:
        return None
    
    indicator_values = state["indicators"][indicator_name]
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