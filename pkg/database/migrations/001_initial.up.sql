-- Create settings table for actor configuration persistence
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(actor_type, actor_id, key)
);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_settings_actor ON settings(actor_type, actor_id);