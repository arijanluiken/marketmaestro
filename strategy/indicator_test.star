# Test strategy for technical indicators
# Updated to use callback-provided data instead of global variables

state = {
    "test_run": False
}

def settings():
    return {
        "interval": "1m",
        "description": "Test strategy for new technical indicators"
    }

def on_kline(kline):
    """Called when a new kline is received"""
    if not state["test_run"]:
        state["test_run"] = True
        return test_indicators()
    
    return {"action": "hold", "reason": "Test completed"}

def test_indicators():
    """
    Test function to verify new indicators work correctly
    """
    
    # Create sample price data
    test_close = [100, 102, 101, 103, 105, 104, 106, 108, 107, 109, 111, 110, 112, 114, 113]
    test_high = [101, 103, 102, 104, 106, 105, 107, 109, 108, 110, 112, 111, 113, 115, 114]
    test_low = [99, 101, 100, 102, 104, 103, 105, 107, 106, 108, 110, 109, 111, 113, 112]
    test_volume = [1000, 1100, 950, 1200, 1300, 1050, 1150, 1400, 1250, 1350, 1500, 1200, 1400, 1600, 1300]
    
    log("Testing new technical indicators...")
    
    # Test OBV
    try:
        obv_result = obv(test_close, test_volume)
        log("✓ OBV calculated successfully. Length: " + str(len(obv_result)))
    except Exception as e:
        log("✗ OBV failed: " + str(e))
    
    # Test ADX
    try:
        adx_result = adx(test_high, test_low, test_close, 10)
        log("✓ ADX calculated successfully")
        if "adx" in adx_result:
            adx_values = adx_result["adx"]
            log("  Last ADX value: " + str(adx_values[-1] if adx_values[-1] else "NaN"))
    except Exception as e:
        log("✗ ADX failed: " + str(e))
    
    # Test Williams %R
    try:
        wr_values = williams_r(test_high, test_low, test_close, 14)
        if wr_values and len(wr_values) > 0:
            current_wr = wr_values[-1]
            if current_wr is not None:
                log("Williams %R: " + str(round(current_wr, 2)))
    except Exception as e:
        log("Williams %R error: " + str(e))
    
    # Test ATR
    try:
        atr_values = atr(test_high, test_low, test_close, 14)
        if atr_values and len(atr_values) > 0:
            current_atr = atr_values[-1]
            if current_atr is not None:
                log("ATR: " + str(round(current_atr, 2)))
    except Exception as e:
        log("ATR error: " + str(e))
    
    log("Indicator tests completed!")
    
    return {
        "action": "hold",
        "reason": "Indicator testing completed successfully"
    }

def on_orderbook(orderbook):
    """Called when orderbook data is received"""
    return {"action": "hold", "reason": "No orderbook signal"}

def on_ticker(ticker):
    """Called when ticker data is received"""
    return {"action": "hold", "reason": "No ticker signal"}
