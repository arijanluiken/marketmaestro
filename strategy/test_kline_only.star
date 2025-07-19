# Test strategy with only on_kline callback
# This strategy demonstrates the callback validation system

def settings():
    return {
        "interval": "1m",
        "test_param": 100
    }

# Only implement on_kline callback - no on_orderbook or on_ticker
def on_kline(kline):
    """This strategy only processes kline data"""
    print("Processing kline for", kline.symbol, "at", kline.timestamp)
    print("Close price:", kline.close)
    
    # Simple logic - always hold
    return {
        "action": "hold",
        "reason": "test strategy - always hold"
    }

# Note: on_orderbook and on_ticker are NOT implemented
# The validation system should detect this and skip calling them
