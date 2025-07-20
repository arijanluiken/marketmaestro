-- Remove the exchange-related changes from settings table
DROP INDEX IF EXISTS idx_settings_unique_key_exchange;
DROP INDEX IF EXISTS idx_settings_exchange;

-- Note: Cannot easily remove the exchange column in SQLite without recreating the table
-- For safety, we'll leave the column but remove the indexes
