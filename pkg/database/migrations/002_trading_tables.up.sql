-- Create orders table for order tracking
CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exchange TEXT NOT NULL,
    order_id TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    type TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL,
    status TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange, order_id)
);

-- Create positions table for position tracking
CREATE TABLE IF NOT EXISTS positions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    size REAL NOT NULL,
    entry_price REAL NOT NULL,
    mark_price REAL NOT NULL,
    unrealized_pl REAL NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange, symbol, side)
);

-- Create portfolio snapshots table for P&L tracking
CREATE TABLE IF NOT EXISTS portfolio_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exchange TEXT NOT NULL,
    total_balance REAL NOT NULL,
    unrealized_pl REAL NOT NULL,
    realized_pl REAL NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create strategy runs table for tracking strategy execution
CREATE TABLE IF NOT EXISTS strategy_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    strategy_name TEXT NOT NULL,
    config TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_orders_exchange_symbol ON orders(exchange, symbol);
CREATE INDEX IF NOT EXISTS idx_positions_exchange_symbol ON positions(exchange, symbol);
CREATE INDEX IF NOT EXISTS idx_portfolio_exchange_time ON portfolio_snapshots(exchange, created_at);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_exchange_symbol ON strategy_runs(exchange, symbol);