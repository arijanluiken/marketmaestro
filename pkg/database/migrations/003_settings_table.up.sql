-- Settings table for storing configuration parameters
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    exchange TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(key, exchange)
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_settings_key_exchange ON settings(key, exchange);
CREATE INDEX IF NOT EXISTS idx_settings_exchange ON settings(exchange);
