# Advanced Volatility-Based Strategy
# Uses ATR, Standard Deviation, and Williams %R for volatility-based trading

# Strategy settings configuration
def settings():
    return {
        "interval": "15m",  # Kline interval for this strategy
        "atr_period": 14,
        "atr_multiplier": 2.0,  # ATR multiplier for stop loss
        "williams_r_period": 14,
        "williams_r_oversold": -80,
        "williams_r_overbought": -20,
        "stddev_period": 20,
        "volatility_threshold": 1.5,  # Standard deviation threshold
        "roc_period": 10,
        "roc_threshold": 5.0,  # ROC percentage threshold for momentum
        "position_size": 0.02,
        "max_risk_percent": 2.0  # Maximum risk per trade
    }

# Strategy state
state = {
    "klines": [],
    "last_signal": "hold",
    "entry_price": 0.0,
    "stop_loss": 0.0,
    "position": None  # "long", "short", or None
}

# Initialize strategy parameters
params = settings()
atr_period = config.get("atr_period", params["atr_period"])
atr_multiplier = config.get("atr_multiplier", params["atr_multiplier"])
williams_r_period = config.get("williams_r_period", params["williams_r_period"])
williams_r_oversold = config.get("williams_r_oversold", params["williams_r_oversold"])
williams_r_overbought = config.get("williams_r_overbought", params["williams_r_overbought"])
stddev_period = config.get("stddev_period", params["stddev_period"])
volatility_threshold = config.get("volatility_threshold", params["volatility_threshold"])
roc_period = config.get("roc_period", params["roc_period"])
roc_threshold = config.get("roc_threshold", params["roc_threshold"])
position_size = config.get("position_size", params["position_size"])
max_risk_percent = config.get("max_risk_percent", params["max_risk_percent"])

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
    max_needed = max(atr_period, williams_r_period, stddev_period, roc_period) + 10
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Extract price data
    closes = [k["close"] for k in state["klines"]]
    highs = [k["high"] for k in state["klines"]]
    lows = [k["low"] for k in state["klines"]]
    
    # Check for trading signals
    return check_volatility_signals(closes, highs, lows)

def check_volatility_signals(closes, highs, lows):
    """Check for volatility-based trading signals"""
    if len(closes) < max(atr_period, williams_r_period, stddev_period, roc_period):
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": closes[-1] if closes else 0,
            "type": "market",
            "reason": "Insufficient data for volatility analysis"
        }
    
    current_price = closes[-1]
    
    # Calculate indicators
    atr_values = atr(highs, lows, closes, atr_period)
    williams_r_values = williams_r(highs, lows, closes, williams_r_period)
    stddev_values = stddev(closes, stddev_period)
    roc_values = roc(closes, roc_period)
    
    # Get current indicator values
    current_atr = get_last_valid_value(atr_values)
    current_williams_r = get_last_valid_value(williams_r_values)
    current_stddev = get_last_valid_value(stddev_values)
    current_roc = get_last_valid_value(roc_values)
    
    if current_atr is None or current_williams_r is None or current_stddev is None or current_roc is None:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": current_price,
            "type": "market",
            "reason": "Indicators not ready"
        }
    
    # Volatility filter - check if market is suitable for trading
    volatility_score = current_stddev / (current_price / 100)  # Normalize volatility as percentage
    
    if volatility_score > volatility_threshold:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": current_price,
            "type": "market",
            "reason": "High volatility - score: " + str(round(volatility_score, 2))
        }
    
    # Position management for existing positions
    if state["position"] is not None:
        return manage_existing_position(current_price, current_atr, current_williams_r)
    
    # Entry signals for new positions
    return check_entry_signals(current_price, current_atr, current_williams_r, current_roc, volatility_score)

def check_entry_signals(current_price, current_atr, current_williams_r, current_roc, volatility_score):
    """Check for new position entry signals"""
    
    # Strong momentum filter using ROC
    strong_bullish_momentum = current_roc > roc_threshold
    strong_bearish_momentum = current_roc < -roc_threshold
    
    # Williams %R oversold/overbought conditions
    williams_oversold = current_williams_r < williams_r_oversold
    williams_overbought = current_williams_r > williams_r_overbought
    
    # Calculate position size based on volatility and risk
    stop_distance = current_atr * atr_multiplier
    risk_amount = current_price * (max_risk_percent / 100)
    volatility_adjusted_size = min(position_size, risk_amount / stop_distance)
    
    # Bullish entry: Williams %R oversold + strong bullish momentum + low volatility
    if williams_oversold and strong_bullish_momentum and volatility_score < volatility_threshold * 0.7:
        state["position"] = "long"
        state["entry_price"] = current_price
        state["stop_loss"] = current_price - stop_distance
        
        reason = "LONG entry: Williams %R oversold (" + str(round(current_williams_r, 1)) + "), strong momentum (ROC: " + str(round(current_roc, 1)) + "%), low volatility (" + str(round(volatility_score, 2)) + ")"
        log("BUY signal: " + reason)
        
        return {
            "action": "buy",
            "quantity": volatility_adjusted_size,
            "price": current_price,
            "type": "market",
            "reason": reason,
            "stop_loss": state["stop_loss"],
            "atr_stop": True
        }
    
    # Bearish entry: Williams %R overbought + strong bearish momentum + low volatility
    elif williams_overbought and strong_bearish_momentum and volatility_score < volatility_threshold * 0.7:
        state["position"] = "short"
        state["entry_price"] = current_price
        state["stop_loss"] = current_price + stop_distance
        
        reason = "SHORT entry: Williams %R overbought (" + str(round(current_williams_r, 1)) + "), strong negative momentum (ROC: " + str(round(current_roc, 1)) + "%), low volatility (" + str(round(volatility_score, 2)) + ")"
        log("SELL signal: " + reason)
        
        return {
            "action": "sell",
            "quantity": volatility_adjusted_size,
            "price": current_price,
            "type": "market",
            "reason": reason,
            "stop_loss": state["stop_loss"],
            "atr_stop": True
        }
    
    # No entry signal
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No entry signal - Williams %R: " + str(round(current_williams_r, 1)) + ", ROC: " + str(round(current_roc, 1)) + "%, Volatility: " + str(round(volatility_score, 2))
    }

def manage_existing_position(current_price, current_atr, current_williams_r):
    """Manage existing positions with dynamic stops"""
    
    if state["position"] == "long":
        # Long position management
        profit_percent = ((current_price - state["entry_price"]) / state["entry_price"]) * 100
        
        # Update trailing stop using ATR
        new_stop = current_price - (current_atr * atr_multiplier)
        if new_stop > state["stop_loss"]:
            state["stop_loss"] = new_stop
            log("Trailing stop updated for LONG: " + str(round(state["stop_loss"], 2)))
        
        # Exit on Williams %R overbought or stop loss hit
        if current_williams_r > williams_r_overbought or current_price <= state["stop_loss"]:
            exit_reason = "Williams %R overbought" if current_williams_r > williams_r_overbought else "Stop loss hit"
            reason = "LONG exit: " + exit_reason + " - Profit: " + str(round(profit_percent, 2)) + "%"
            log("SELL signal: " + reason)
            
            # Reset position state
            state["position"] = None
            state["entry_price"] = 0.0
            state["stop_loss"] = 0.0
            
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": reason
            }
    
    elif state["position"] == "short":
        # Short position management
        profit_percent = ((state["entry_price"] - current_price) / state["entry_price"]) * 100
        
        # Update trailing stop using ATR
        new_stop = current_price + (current_atr * atr_multiplier)
        if new_stop < state["stop_loss"]:
            state["stop_loss"] = new_stop
            log("Trailing stop updated for SHORT: " + str(round(state["stop_loss"], 2)))
        
        # Exit on Williams %R oversold or stop loss hit
        if current_williams_r < williams_r_oversold or current_price >= state["stop_loss"]:
            exit_reason = "Williams %R oversold" if current_williams_r < williams_r_oversold else "Stop loss hit"
            reason = "SHORT exit: " + exit_reason + " - Profit: " + str(round(profit_percent, 2)) + "%"
            log("BUY signal: " + reason)
            
            # Reset position state
            state["position"] = None
            state["entry_price"] = 0.0
            state["stop_loss"] = 0.0
            
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": reason
            }
    
    # Hold current position
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "Managing " + state["position"] + " position - Stop: " + str(round(state["stop_loss"], 2))
    }

def get_last_valid_value(values):
    """Get the last non-None value from an indicator array"""
    if not values or len(values) == 0:
        return None
    
    # Look for the last valid (non-None) value
    for i in range(len(values) - 1, -1, -1):
        if values[i] is not None:
            return values[i]
    
    return None

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    # Use tight spread requirement for volatility strategy
    if len(orderbook.bids) > 0 and len(orderbook.asks) > 0:
        spread = orderbook.asks[0].price - orderbook.bids[0].price
        mid_price = (orderbook.bids[0].price + orderbook.asks[0].price) / 2
        spread_percent = (spread / mid_price) * 100
        
        # Tighter spread requirement for volatility-based entries
        if spread_percent > 0.15:
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": 0.0,
                "type": "market",
                "reason": "Spread too wide for volatility strategy: " + str(round(spread_percent, 3)) + "%"
            }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "Orderbook conditions acceptable for volatility strategy"
    }

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": ticker.price,
        "type": "market",
        "reason": "Volatility strategy ticker processing"
    }