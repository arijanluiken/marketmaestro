# Enhanced Order Management Strategy Example

# Strategy settings configuration with advanced order management
def settings():
    return {
        "interval": "1m",  # Default kline interval - can be overridden in config
        "short_period": 10,
        "long_period": 20,
        "position_size": 0.01,
        "use_trailing_stops": True,
        "trail_percent": 2.0,  # 2% trailing stop
        "stop_loss_percent": 1.5,  # 1.5% stop loss
        "take_profit_percent": 3.0  # 3% take profit
    }

# Initialize strategy parameters - these will be set from config in callbacks
params = settings()
short_period = params["short_period"]  # Default values, will be overridden by config
long_period = params["long_period"]
position_size = params["position_size"]
use_trailing_stops = params["use_trailing_stops"]
trail_percent = params["trail_percent"]
stop_loss_percent = params["stop_loss_percent"]
take_profit_percent = params["take_profit_percent"]

def get_config_values():
    """Get configuration values with fallbacks to defaults"""
    return {
        "short_period": get_config("short_period", params["short_period"]),
        "long_period": get_config("long_period", params["long_period"]),
        "position_size": get_config("position_size", params["position_size"]),
        "use_trailing_stops": get_config("use_trailing_stops", params["use_trailing_stops"]),
        "trail_percent": get_config("trail_percent", params["trail_percent"]),
        "stop_loss_percent": get_config("stop_loss_percent", params["stop_loss_percent"]),
        "take_profit_percent": get_config("take_profit_percent", params["take_profit_percent"])
    }

def init_state():
    """Initialize strategy state using thread-safe state management"""
    # Initialize state values if they don't exist
    if get_state("initialized", False) == False:
        set_state("initialized", True)
        set_state("short_ma", [])
        set_state("long_ma", [])
        set_state("klines", [])
        set_state("last_signal", "hold")
        set_state("active_position", None)
        set_state("entry_price", 0.0)

def on_kline(kline):
    """Called when a new kline is received"""
    # Initialize state if needed
    init_state()
    
    # Get config values from runtime context
    cfg = get_config_values()
    short_period = cfg["short_period"]
    long_period = cfg["long_period"]
    position_size = cfg["position_size"]
    use_trailing_stops = cfg["use_trailing_stops"]
    trail_percent = cfg["trail_percent"]
    stop_loss_percent = cfg["stop_loss_percent"]
    take_profit_percent = cfg["take_profit_percent"]
    
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
    max_needed = max(short_period, long_period) + 10
    if len(klines) > max_needed:
        klines = klines[-max_needed:]
    
    # Update state
    set_state("klines", klines)
    
    # Extract close prices
    closes = [k["close"] for k in klines]
    
    # Calculate moving averages
    if len(closes) >= short_period:
        short_ma = sma(closes, short_period)
        set_state("short_ma", short_ma)
    
    if len(closes) >= long_period:
        long_ma = sma(closes, long_period)
        set_state("long_ma", long_ma)
    
    # Check for trading signals
    return check_signals(closes, cfg)

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

def check_signals(closes, cfg):
    """Check for crossover signals with advanced order management"""
    short_ma = get_state("short_ma", [])
    long_ma = get_state("long_ma", [])
    
    if len(short_ma) < 2 or len(long_ma) < 2:
        return {
            "action": "hold",
            "quantity": 0.0,
            "price": 0.0,
            "type": "market", 
            "reason": "Insufficient data for signal"
        }
    
    # Get current and previous values
    current_short = short_ma[-1]
    current_long = long_ma[-1]
    prev_short = short_ma[-2]
    prev_long = long_ma[-2]
    current_price = closes[-1]
    
    # Check for bullish crossover (short MA crosses above long MA)
    if prev_short <= prev_long and current_short > current_long:
        reason = "Short MA (" + str(round(current_short, 2)) + ") crossed above Long MA (" + str(round(current_long, 2)) + ")"
        log("BUY signal: " + reason)
        set_state("last_signal", "buy")
        set_state("active_position", "long")
        set_state("entry_price", current_price)
        
        if cfg["use_trailing_stops"]:
            # Use trailing stop for dynamic risk management
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
                "price": current_price,
                "type": "market",
                "reason": reason,
                "trail_percent": cfg["trail_percent"],
                "stop_loss": "trailing"
            }
        else:
            # Use fixed stop loss
            stop_price = current_price * (1 - cfg["stop_loss_percent"] / 100)
            take_profit_price = current_price * (1 + cfg["take_profit_percent"] / 100)
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
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
        set_state("last_signal", "sell")
        set_state("active_position", "short")
        set_state("entry_price", current_price)
        
        if cfg["use_trailing_stops"]:
            # Use trailing stop for dynamic risk management
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "price": current_price,
                "type": "market",
                "reason": reason,
                "trail_percent": cfg["trail_percent"],
                "stop_loss": "trailing"
            }
        else:
            # Use fixed stop loss
            stop_price = current_price * (1 + cfg["stop_loss_percent"] / 100)  # Stop above for short
            take_profit_price = current_price * (1 - cfg["take_profit_percent"] / 100)
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
                "price": current_price,
                "type": "market",
                "reason": reason,
                "stop_price": stop_price,
                "take_profit": take_profit_price,
                "stop_loss": "fixed"
            }
    
    # Check for position management signals
    active_position = get_state("active_position", None)
    entry_price = get_state("entry_price", 0.0)
    if active_position and entry_price > 0:
        return check_position_management(current_price, cfg)
    
    # No signal
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "No crossover detected"
    }

def check_position_management(current_price, cfg):
    """Check if we need to manage existing positions"""
    active_position = get_state("active_position", None)
    entry_price = get_state("entry_price", 0.0)
    
    if active_position == "long":
        # Check if we're in profit and should tighten stops
        profit_percent = ((current_price - entry_price) / entry_price) * 100
        
        if profit_percent > cfg["take_profit_percent"]:
            log("Taking profit at " + str(round(profit_percent, 2)) + "% gain")
            set_state("active_position", None)
            set_state("entry_price", 0.0)
            return {
                "action": "sell",
                "quantity": cfg["position_size"],
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
                "trail_percent": cfg["trail_percent"] * 0.5  # Tighten to 1%
            }
    
    elif active_position == "short":
        # Check short position management
        profit_percent = ((entry_price - current_price) / entry_price) * 100
        
        if profit_percent > cfg["take_profit_percent"]:
            log("Taking profit on short at " + str(round(profit_percent, 2)) + "% gain")
            set_state("active_position", None)
            set_state("entry_price", 0.0)
            return {
                "action": "buy",
                "quantity": cfg["position_size"],
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
                "trail_percent": cfg["trail_percent"] * 0.5  # Tighten to 1%
            }
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": current_price,
        "type": "market",
        "reason": "Position management - no action needed"
    }
