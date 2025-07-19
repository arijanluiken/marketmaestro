# Complete Strategy Example with Lifecycle Callbacks
# Demonstrates on_start, on_stop, and trading logic

def settings():
    return {
        "interval": "5m",
        "symbol": "BTCUSDT",
        "description": "Complete strategy with lifecycle management"
    }

def on_start():
    """
    Strategy initialization callback.
    Called once when the strategy starts.
    """
    print("🚀 Strategy Started")
    print("📊 Initializing technical analysis parameters")
    print("⚙️  Strategy configuration loaded")
    print("✅ Ready to process market data")

def on_stop():
    """
    Strategy cleanup callback.
    Called once when the strategy stops.
    """
    print("🛑 Strategy Stopping")
    print("💾 Saving final state")
    print("🧹 Cleaning up resources")
    print("✅ Strategy stopped cleanly")

def on_kline(kline):
    """
    Main trading logic called for each new kline.
    """
    # Get historical data
    high = [float(k.high) for k in get_historical_klines(50)]
    low = [float(k.low) for k in get_historical_klines(50)]
    close = [float(k.close) for k in get_historical_klines(50)]
    volume = [float(k.volume) for k in get_historical_klines(50)]
    
    if len(close) < 50:
        return
    
    current_price = float(kline.close)
    
    # Multi-indicator analysis
    # 1. Trend Analysis
    supertrend_data = supertrend(high, low, close, 10, 3.0)
    current_trend = supertrend_data["trend"][-1] if len(supertrend_data["trend"]) > 0 else 0
    
    # 2. Momentum Analysis
    rsi_values = rsi(close, 14)
    current_rsi = rsi_values[-1] if len(rsi_values) > 0 else 50
    
    stoch_rsi = stochastic_rsi(close, 14, 3, 3)
    stoch_k = stoch_rsi["k"][-1] if len(stoch_rsi["k"]) > 0 else 50
    stoch_d = stoch_rsi["d"][-1] if len(stoch_rsi["d"]) > 0 else 50
    
    # 3. Volume Analysis
    vwap_values = vwap(high, low, close, volume)
    current_vwap = vwap_values[-1] if len(vwap_values) > 0 else current_price
    
    # 4. Volatility Analysis
    heikin_ashi_data = heikin_ashi_candles([float(k.open) for k in get_historical_klines(50)], high, low, close)
    ha_close = heikin_ashi_data["close"][-1] if len(heikin_ashi_data["close"]) > 0 else current_price
    ha_open = heikin_ashi_data["open"][-1] if len(heikin_ashi_data["open"]) > 0 else current_price
    
    # Trading Logic
    # Long Entry: Bullish trend + oversold momentum + price above VWAP + bullish Heikin Ashi
    long_signal = (
        current_trend == 1 and  # Bullish Supertrend
        current_rsi < 40 and    # Oversold RSI
        stoch_k < 20 and        # Oversold Stochastic RSI
        current_price > current_vwap and  # Above VWAP
        ha_close > ha_open      # Bullish Heikin Ashi candle
    )
    
    # Short Entry: Bearish trend + overbought momentum + price below VWAP + bearish Heikin Ashi
    short_signal = (
        current_trend == -1 and  # Bearish Supertrend
        current_rsi > 60 and     # Overbought RSI
        stoch_k > 80 and         # Overbought Stochastic RSI
        current_price < current_vwap and  # Below VWAP
        ha_close < ha_open       # Bearish Heikin Ashi candle
    )
    
    # Execute trades
    if long_signal:
        print("🔵 LONG SIGNAL at " + str(current_price))
        print("   Trend: " + str(current_trend) + " | RSI: " + str(current_rsi) + " | Stoch K: " + str(stoch_k))
        signal("buy", 0.02)
    elif short_signal:
        print("🔴 SHORT SIGNAL at " + str(current_price))
        print("   Trend: " + str(current_trend) + " | RSI: " + str(current_rsi) + " | Stoch K: " + str(stoch_k))
        signal("sell", 0.02)
    else:
        # Log current market state every 10th kline to avoid spam
        if int(current_price) % 10 == 0:
            print("📊 Market Analysis - Price: " + str(current_price) + " | Trend: " + str(current_trend) + " | RSI: " + str(current_rsi))

def on_orderbook(orderbook):
    """
    Called when orderbook data is received.
    Can be used for advanced order flow analysis.
    """
    # Example: Monitor large bid/ask levels
    pass

def on_ticker(ticker):
    """
    Called when ticker data is received.
    Can be used for volume or price alerts.
    """
    # Example: Volume spike detection
    if float(ticker.volume) > 1000000:  # Example threshold
        print("📈 High volume detected: " + str(ticker.volume))
