# Test script for new technical indicators

def test_indicators():
    """
    Test function to verify new indicators work correctly
    """
    
    # Create sample price data
    test_close = [100, 102, 101, 103, 105, 104, 106, 108, 107, 109, 111, 110, 112, 114, 113]
    test_high = [101, 103, 102, 104, 106, 105, 107, 109, 108, 110, 112, 111, 113, 115, 114]
    test_low = [99, 101, 100, 102, 104, 103, 105, 107, 106, 108, 110, 109, 111, 113, 112]
    test_volume = [1000, 1100, 950, 1200, 1300, 1050, 1150, 1400, 1250, 1350, 1500, 1200, 1400, 1600, 1300]
    
    print("Testing new technical indicators...")
    
    # Test OBV
    try:
        obv_result = obv(test_close, test_volume)
        print(f"✓ OBV calculated successfully. Length: {len(obv_result)}")
        print(f"  Last 3 OBV values: {obv_result[-3:]}")
    except Exception as e:
        print(f"✗ OBV failed: {e}")
    
    # Test ADX
    try:
        adx_result = adx(test_high, test_low, test_close, 10)
        print(f"✓ ADX calculated successfully")
        print(f"  ADX keys: {list(adx_result.keys())}")
        if "adx" in adx_result:
            adx_values = adx_result["adx"]
            print(f"  Last ADX value: {adx_values[-1] if adx_values[-1] else 'NaN'}")
    except Exception as e:
        print(f"✗ ADX failed: {e}")
    
    # Test Parabolic SAR
    try:
        psar_result = parabolic_sar(test_high, test_low)
        print(f"✓ Parabolic SAR calculated successfully. Length: {len(psar_result)}")
        print(f"  Last 3 PSAR values: {psar_result[-3:]}")
    except Exception as e:
        print(f"✗ Parabolic SAR failed: {e}")
    
    # Test Keltner Channels
    try:
        keltner_result = keltner(test_high, test_low, test_close, 10)
        print(f"✓ Keltner Channels calculated successfully")
        print(f"  Keltner keys: {list(keltner_result.keys())}")
        if "middle" in keltner_result:
            middle_values = keltner_result["middle"]
            print(f"  Last middle value: {middle_values[-1] if middle_values[-1] else 'NaN'}")
    except Exception as e:
        print(f"✗ Keltner Channels failed: {e}")
    
    # Test Ichimoku
    try:
        ichimoku_result = ichimoku(test_high, test_low, test_close)
        print(f"✓ Ichimoku calculated successfully")
        print(f"  Ichimoku keys: {list(ichimoku_result.keys())}")
        if "tenkan_sen" in ichimoku_result:
            tenkan_values = ichimoku_result["tenkan_sen"]
            print(f"  Last Tenkan-sen value: {tenkan_values[-1] if tenkan_values[-1] else 'NaN'}")
    except Exception as e:
        print(f"✗ Ichimoku failed: {e}")
    
    # Test Pivot Points
    try:
        pivot_result = pivot_points(test_high, test_low, test_close)
        print(f"✓ Pivot Points calculated successfully")
        print(f"  Pivot keys: {list(pivot_result.keys())}")
        if "pivot" in pivot_result:
            pivot_values = pivot_result["pivot"]
            print(f"  Last pivot value: {pivot_values[-1] if pivot_values[-1] else 'NaN'}")
    except Exception as e:
        print(f"✗ Pivot Points failed: {e}")
    
    # Test Fibonacci
    try:
        fib_result = fibonacci(115.0, 99.0)
        print(f"✓ Fibonacci calculated successfully")
        print(f"  Fibonacci keys: {list(fib_result.keys())}")
        print(f"  50% level: {fib_result['50.0']}")
        print(f"  61.8% level: {fib_result['61.8']}")
    except Exception as e:
        print(f"✗ Fibonacci failed: {e}")
    
    # Test Aroon
    try:
        aroon_result = aroon(test_high, test_low, 10)
        print(f"✓ Aroon calculated successfully")
        print(f"  Aroon keys: {list(aroon_result.keys())}")
        if "aroon_up" in aroon_result:
            aroon_up_values = aroon_result["aroon_up"]
            print(f"  Last Aroon Up value: {aroon_up_values[-1] if aroon_up_values[-1] else 'NaN'}")
    except Exception as e:
        print(f"✗ Aroon failed: {e}")
    
    print("\\nIndicator testing completed!")
    
    return {"action": "hold", "reason": "Test completed successfully"}

def get_signal(data):
    """
    Main strategy function - runs tests
    """
    return test_indicators()

# Configuration
config = {
    "symbol": "BTCUSDT",
    "interval": "1m", 
    "description": "Test strategy for new technical indicators"
}
    
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