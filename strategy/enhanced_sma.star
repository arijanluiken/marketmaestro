# Enhanced Order Management Strategy Example

# Strategy settings configuration with advanced order management
def settings():
    return {
        "interval": "1m",  # Kline interval for this strategy
        "short_period": 10,
        "long_period": 20,
        "position_size": 0.01,
        "use_trailing_stops": True,
        "trail_percent": 2.0,  # 2% trailing stop
        "stop_loss_percent": 1.5,  # 1.5% stop loss
        "take_profit_percent": 3.0  # 3% take profit
    }

# Strategy state
state = {
    "short_ma": [],
    "long_ma": [],
    "klines": [],
    "last_signal": "hold",
    "active_position": None,
    "entry_price": 0.0
}

# Initialize strategy parameters
params = settings()
short_period = config.get("short_period", params["short_period"])
long_period = config.get("long_period", params["long_period"])
position_size = config.get("position_size", params["position_size"])
use_trailing_stops = config.get("use_trailing_stops", params["use_trailing_stops"])
trail_percent = config.get("trail_percent", params["trail_percent"])
stop_loss_percent = config.get("stop_loss_percent", params["stop_loss_percent"])
take_profit_percent = config.get("take_profit_percent", params["take_profit_percent"])

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
    max_needed = max(short_period, long_period) + 10
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Extract close prices
    closes = [k["close"] for k in state["klines"]]
    
    # Calculate moving averages
    if len(closes) >= short_period:
        state["short_ma"] = sma(closes, short_period)
    
    if len(closes) >= long_period:
        state["long_ma"] = sma(closes, long_period)
    
    # Check for trading signals
    return check_signals(closes)

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    # Use spread analysis for entry timing
    if len(orderbook.bids) > 0 and len(orderbook.asks) > 0:
        spread = orderbook.asks[0].price - orderbook.bids[0].price
        mid_price = (orderbook.bids[0].price + orderbook.asks[0].price) / 2
        spread_percent = (spread / mid_price) * 100
        
        # Only trade when spread is reasonable (less than 0.1%)
        if spread_percent > 0.1:
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": 0.0,
                "type": "market",
                "reason": "Spread too wide: " + str(round(spread_percent, 3)) + "%"
            }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "No orderbook signal"
    }

def on_ticker(ticker):
    """Called when ticker data is received"""
    # Use volume analysis for signal confirmation
    if ticker.volume > 0:
        # Higher volume could confirm trend strength
        volume_factor = min(ticker.volume / 1000.0, 2.0)  # Cap at 2x
        log("Volume factor: " + str(round(volume_factor, 2)))
        
        # Could adjust position size based on volume
        if volume_factor > 1.5:
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": ticker.price,
                "type": "market",
                "reason": "High volume confirmation"
            }
    
    return {
        "action": "hold", 
        "quantity": 0.0,
        "price": ticker.price,
        "type": "market",
        "reason": "No ticker signal"
    }

def check_signals(closes):
    """Check for crossover signals with advanced order management"""
    if len(state["short_ma"]) < 2 or len(state["long_ma"]) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market", 
            "reason": "Insufficient data for signal"
        }
    
    # Get current and previous values
    current_short = state["short_ma"][-1]
    current_long = state["long_ma"][-1]
    prev_short = state["short_ma"][-2]
    prev_long = state["long_ma"][-2]
    current_price = closes[-1]
    
    # Check for bullish crossover (short MA crosses above long MA)
    if prev_short <= prev_long and current_short > current_long:
        reason = "Short MA (" + str(round(current_short, 2)) + ") crossed above Long MA (" + str(round(current_long, 2)) + ")"
        log("BUY signal: " + reason)
        state["last_signal"] = "buy"
        state["active_position"] = "long"
        state["entry_price"] = current_price
        
        if use_trailing_stops:
            # Use trailing stop for dynamic risk management
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": reason,
                "trail_percent": trail_percent,
                "stop_loss": "trailing"
            }
        else:
            # Use fixed stop loss
            stop_price = current_price * (1 - stop_loss_percent / 100)
            take_profit_price = current_price * (1 + take_profit_percent / 100)
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": reason,
                "stop_price": stop_price,
                "take_profit": take_profit_price,
                "stop_loss": "fixed"
            }
    
    # Check for bearish crossover (short MA crosses below long MA)
    elif prev_short >= prev_long and current_short < current_long:
        reason = "Short MA (" + str(round(current_short, 2)) + ") crossed below Long MA (" + str(round(current_long, 2)) + ")"
        log("SELL signal: " + reason)
        state["last_signal"] = "sell"
        state["active_position"] = "short"
        state["entry_price"] = current_price
        
        if use_trailing_stops:
            # Use trailing stop for dynamic risk management
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": reason,
                "trail_percent": trail_percent,
                "stop_loss": "trailing"
            }
        else:
            # Use fixed stop loss
            stop_price = current_price * (1 + stop_loss_percent / 100)  # Stop above for short
            take_profit_price = current_price * (1 - take_profit_percent / 100)
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": reason,
                "stop_price": stop_price,
                "take_profit": take_profit_price,
                "stop_loss": "fixed"
            }
    
    # Check for position management signals
    if state["active_position"] and state["entry_price"] > 0:
        return check_position_management(current_price)
    
    # No signal
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No crossover detected"
    }

def check_position_management(current_price):
    """Check if we need to manage existing positions"""
    if state["active_position"] == "long":
        # Check if we're in profit and should tighten stops
        profit_percent = ((current_price - state["entry_price"]) / state["entry_price"]) * 100
        
        if profit_percent > take_profit_percent:
            log("Taking profit at " + str(round(profit_percent, 2)) + "% gain")
            state["active_position"] = None
            state["entry_price"] = 0.0
            return {
                "action": "sell",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Take profit at " + str(round(profit_percent, 2)) + "% gain"
            }
        
        elif profit_percent > 1.0:  # In profit, tighten trailing stop
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": current_price,
                "type": "market",
                "reason": "Tightening trailing stop - profit: " + str(round(profit_percent, 2)) + "%",
                "trail_percent": trail_percent * 0.5  # Tighten to 1%
            }
    
    elif state["active_position"] == "short":
        # Check short position management
        profit_percent = ((state["entry_price"] - current_price) / state["entry_price"]) * 100
        
        if profit_percent > take_profit_percent:
            log("Taking profit on short at " + str(round(profit_percent, 2)) + "% gain")
            state["active_position"] = None
            state["entry_price"] = 0.0
            return {
                "action": "buy",
                "quantity": position_size,
                "price": current_price,
                "type": "market",
                "reason": "Take profit on short at " + str(round(profit_percent, 2)) + "% gain"
            }
        
        elif profit_percent > 1.0:  # In profit, tighten trailing stop
            return {
                "action": "hold",
                "quantity": 0.0,
                "price": current_price,
                "type": "market",
                "reason": "Tightening trailing stop on short - profit: " + str(round(profit_percent, 2)) + "%",
                "trail_percent": trail_percent * 0.5  # Tighten to 1%
            }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "Position management - no action needed"
    }
