-- Add exchange column to existing settings table
-- SQLite doesn't support adding NOT NULL columns to existing tables with data
-- So we'll add it as nullable first, then update all rows to have default value

-- Add exchange column as nullable initially
ALTER TABLE settings ADD COLUMN exchange TEXT DEFAULT '';

-- Update all existing rows to have empty exchange value
UPDATE settings SET exchange = '' WHERE exchange IS NULL;

-- Drop the old unique constraint based on actor_type/actor_id if it exists
DROP INDEX IF EXISTS idx_settings_actor;

-- Create new indexes for the updated schema
CREATE INDEX IF NOT EXISTS idx_settings_key_exchange ON settings(key, exchange);
CREATE INDEX IF NOT EXISTS idx_settings_exchange ON settings(exchange);
