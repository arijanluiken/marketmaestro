# Test Strategy to Validate Updated Documentation
# This strategy tests the patterns described in the updated strategy-engine.md

def settings():
    return {
        "interval": "1m",
        "sma_period": 10,
        "rsi_period": 14,
        "position_size": 0.01
    }

# Strategy state to maintain kline buffer as documented
state = {
    "klines": [],
    "position": 0
}

# Initialize strategy parameters like working strategies do
params = settings()
sma_period = config.get("sma_period", params["sma_period"])
rsi_period = config.get("rsi_period", params["rsi_period"])
position_size = config.get("position_size", params["position_size"])

def on_start():
    """Test the on_start callback"""
    print("🚀 Test strategy starting")
    print("📊 Testing updated documentation patterns")
    
def on_stop():
    """Test the on_stop callback"""
    print("🛑 Test strategy stopping")
    print("📋 Documentation validation complete")

def on_kline(kline):
    """Test the callback-based pattern with internal buffer"""
    # Maintain internal kline buffer as documented
    state["klines"].append({
        "timestamp": kline.timestamp,
        "open": kline.open,
        "high": kline.high,
        "low": kline.low,
        "close": kline.close,
        "volume": kline.volume
    })
    
    # Keep buffer manageable using module-level parameters
    max_needed = max(sma_period, rsi_period) + 10
    
    if len(state["klines"]) > max_needed:
        state["klines"] = state["klines"][-max_needed:]
    
    # Data validation as documented
    if len(state["klines"]) < rsi_period:
        return {"action": "hold", "reason": "Insufficient data for calculations"}
    
    # Extract price arrays from internal buffer as documented
    close_prices = [k["close"] for k in state["klines"]]
    high_prices = [k["high"] for k in state["klines"]]
    low_prices = [k["low"] for k in state["klines"]]
    
    # Calculate indicators using extracted data
    sma_values = sma(close_prices, sma_period)
    rsi_values = rsi(close_prices, rsi_period)
    
    # Test attribute access on kline object
    current_price = kline.close  # Using attribute access as documented
    current_sma = sma_values[-1]
    current_rsi = rsi_values[-1]
    
    # Simple strategy logic to test signal return format
    if current_rsi < 30 and current_price > current_sma:
        log("BUY signal: RSI " + str(round(current_rsi, 1)) + " oversold, price " + str(current_price) + " above SMA " + str(round(current_sma, 2)))
        return {
            "action": "buy",
            "quantity": position_size,
            "type": "market",
            "reason": "RSI oversold at " + str(round(current_rsi, 1))
        }
    
    elif current_rsi > 70 and current_price < current_sma:
        log("SELL signal: RSI " + str(round(current_rsi, 1)) + " overbought, price " + str(current_price) + " below SMA " + str(round(current_sma, 2)))
        return {
            "action": "sell", 
            "quantity": position_size,
            "type": "market",
            "reason": "RSI overbought at " + str(round(current_rsi, 1))
        }
    
    return {
        "action": "hold",
        "reason": "Waiting - RSI: " + str(round(current_rsi, 1)) + ", Price vs SMA: " + str(round((current_price/current_sma - 1)*100, 1)) + "%"
    }

def on_orderbook(orderbook):
    """Test orderbook callback"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Test ticker callback"""
    return {"action": "hold", "reason": "No ticker signal"}