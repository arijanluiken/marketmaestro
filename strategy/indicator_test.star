# Simple indicator test strategy
# Tests new indicators with minimal data

# Strategy settings
def settings():
    return {
        "interval": "1m",
        "position_size": 0.01
    }

# Basic test function to verify new indicators are working
def on_kline(kline):
    """Test new indicators with sample data"""
    
    # Sample price data for testing
    test_closes = [100, 101, 102, 103, 104, 103, 102, 101, 100, 99, 98, 97, 98, 99, 100, 101, 102, 103, 104, 105]
    test_highs = [101, 102, 103, 104, 105, 104, 103, 102, 101, 100, 99, 98, 99, 100, 101, 102, 103, 104, 105, 106]
    test_lows = [99, 100, 101, 102, 103, 102, 101, 100, 99, 98, 97, 96, 97, 98, 99, 100, 101, 102, 103, 104]
    test_volumes = [1000, 1100, 1200, 1300, 1400, 1350, 1250, 1150, 1050, 950, 850, 750, 850, 950, 1050, 1150, 1250, 1350, 1450, 1550]
    
    log("Testing new indicators...")
    
    # Test Williams %R
    try:
        wr_values = williams_r(test_highs, test_lows, test_closes, 14)
        if wr_values and len(wr_values) > 0:
            current_wr = wr_values[-1]
            if current_wr is not None:
                log("Williams %R: " + str(round(current_wr, 2)))
    except Exception as e:
        log("Williams %R error: " + str(e))
    
    # Test ATR
    try:
        atr_values = atr(test_highs, test_lows, test_closes, 14)
        if atr_values and len(atr_values) > 0:
            current_atr = atr_values[-1]
            if current_atr is not None:
                log("ATR: " + str(round(current_atr, 2)))
    except Exception as e:
        log("ATR error: " + str(e))
    
    # Test CCI
    try:
        cci_values = cci(test_highs, test_lows, test_closes, 14)
        if cci_values and len(cci_values) > 0:
            current_cci = cci_values[-1]
            if current_cci is not None:
                log("CCI: " + str(round(current_cci, 2)))
    except Exception as e:
        log("CCI error: " + str(e))
    
    # Test VWAP
    try:
        vwap_values = vwap(test_highs, test_lows, test_closes, test_volumes)
        if vwap_values and len(vwap_values) > 0:
            current_vwap = vwap_values[-1]
            if current_vwap is not None:
                log("VWAP: " + str(round(current_vwap, 2)))
    except Exception as e:
        log("VWAP error: " + str(e))
    
    # Test MFI
    try:
        mfi_values = mfi(test_highs, test_lows, test_closes, test_volumes, 14)
        if mfi_values and len(mfi_values) > 0:
            current_mfi = mfi_values[-1]
            if current_mfi is not None:
                log("MFI: " + str(round(current_mfi, 2)))
    except Exception as e:
        log("MFI error: " + str(e))
    
    # Test Standard Deviation
    try:
        stddev_values = stddev(test_closes, 10)
        if stddev_values and len(stddev_values) > 0:
            current_stddev = stddev_values[-1]
            if current_stddev is not None:
                log("StdDev: " + str(round(current_stddev, 2)))
    except Exception as e:
        log("StdDev error: " + str(e))
    
    # Test Rate of Change
    try:
        roc_values = roc(test_closes, 10)
        if roc_values and len(roc_values) > 0:
            current_roc = roc_values[-1]
            if current_roc is not None:
                log("ROC: " + str(round(current_roc, 2)) + "%")
    except Exception as e:
        log("ROC error: " + str(e))
    
    log("Indicator test completed")
    
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": kline.close,
        "type": "market",
        "reason": "Indicator test strategy - all indicators tested"
    }

def on_orderbook(orderbook):
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": 0.0,
        "type": "market",
        "reason": "Test strategy - no orderbook action"
    }

def on_ticker(ticker):
    return {
        "action": "hold",
        "quantity": 0.0,
        "price": ticker.price,
        "type": "market",
        "reason": "Test strategy - no ticker action"
    }