# Equal Weight Rebalancing Strategy
# This script rebalances the portfolio to maintain equal weight allocation

def settings():
    """Configure rebalancing parameters"""
    return {
        "rebalance_interval": "1h",     # How often to check for rebalancing
        "target_allocation": {          # Target allocations (must sum to 1.0)
            "BTC": 0.40,               # 40% Bitcoin
            "ETH": 0.30,               # 30% Ethereum  
            "SOL": 0.20,               # 20% Solana
            "CASH": 0.10               # 10% Cash
        },
        "rebalance_threshold": 0.05,    # Rebalance when drift > 5%
        "min_trade_amount": 10.0,       # Minimum trade amount in USD
        "max_trades_per_rebalance": 5   # Maximum trades per rebalancing session
    }

def on_rebalance():
    """Main rebalancing logic - called periodically"""
    log("🔄 Starting portfolio rebalancing")
    
    # Get current portfolio state
    current_balances = get_balances()
    current_prices = get_current_prices()
    total_value = get_portfolio_value()
    
    if total_value < 100:  # Skip if portfolio too small
        log("💰 Portfolio too small for rebalancing")
        return {"action": "skip", "reason": "Portfolio value below minimum"}
    
    # Calculate current allocations
    current_allocations = calculate_current_allocations(current_balances, current_prices, total_value)
    target_allocations = config.get("target_allocation", {})
    
    log(f"📊 Portfolio Value: ${total_value:.2f}")
    log(f"📈 Current Allocations: {current_allocations}")
    log(f"🎯 Target Allocations: {target_allocations}")
    
    # Calculate required trades
    trades = calculate_rebalancing_trades(
        current_allocations, 
        target_allocations, 
        total_value,
        current_prices
    )
    
    if not trades:
        log("✅ Portfolio is already balanced")
        return {"action": "hold", "reason": "Portfolio within threshold"}
    
    # Execute trades
    executed_trades = []
    max_trades = config.get("max_trades_per_rebalance", 5)
    
    for i, trade in enumerate(trades[:max_trades]):
        if execute_trade(trade):
            executed_trades.append(trade)
            log(f"✅ Executed trade {i+1}: {trade}")
        else:
            log(f"❌ Failed to execute trade {i+1}: {trade}")
    
    return {
        "action": "rebalanced",
        "trades_executed": len(executed_trades),
        "total_trades_planned": len(trades),
        "reason": f"Rebalanced portfolio with {len(executed_trades)} trades"
    }

def calculate_current_allocations(balances, prices, total_value):
    """Calculate current allocation percentages"""
    allocations = {}
    
    for asset, balance in balances.items():
        if asset == "USDT" or asset == "USD":
            asset_value = balance
        else:
            price = prices.get(asset + "USDT", prices.get(asset + "USD", 0))
            asset_value = balance * price
        
        allocations[asset] = asset_value / total_value if total_value > 0 else 0
    
    return allocations

def calculate_rebalancing_trades(current, target, total_value, prices):
    """Calculate trades needed to rebalance portfolio"""
    trades = []
    threshold = config.get("rebalance_threshold", 0.05)
    min_trade = config.get("min_trade_amount", 10.0)
    
    for asset, target_pct in target.items():
        current_pct = current.get(asset, 0)
        drift = abs(current_pct - target_pct)
        
        if drift > threshold:
            target_value = target_pct * total_value
            current_value = current_pct * total_value
            trade_value = target_value - current_value
            
            if abs(trade_value) >= min_trade:
                if asset == "CASH" or asset == "USDT":
                    # Cash adjustment handled through other asset trades
                    continue
                    
                price = prices.get(asset + "USDT", prices.get(asset + "USD", 0))
                if price > 0:
                    quantity = abs(trade_value) / price
                    side = "buy" if trade_value > 0 else "sell"
                    
                    trades.append({
                        "symbol": asset + "USDT",
                        "side": side,
                        "quantity": round(quantity, 6),
                        "type": "market",
                        "reason": f"Rebalance {asset}: {current_pct:.1%} → {target_pct:.1%}"
                    })
    
    # Sort trades by value (largest first)
    trades.sort(key=lambda t: t["quantity"], reverse=True)
    return trades

def execute_trade(trade):
    """Execute a single rebalancing trade"""
    try:
        result = place_order(
            symbol=trade["symbol"],
            side=trade["side"],
            quantity=trade["quantity"],
            order_type=trade["type"],
            reason=trade["reason"]
        )
        
        if result.get("success"):
            log(f"✅ Trade executed: {trade['side']} {trade['quantity']} {trade['symbol']}")
            return True
        else:
            log(f"❌ Trade failed: {result.get('error', 'Unknown error')}")
            return False
            
    except Exception as e:
        log(f"❌ Trade exception: {str(e)}")
        return False

# Helper function for logging
def log(message):
    """Log messages with timestamp"""
    print(f"[REBALANCE] {message}")
