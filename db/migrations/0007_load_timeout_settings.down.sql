-- Drop timeout column
ALTER TABLE gameserver_settings
DROP COLUMN IF EXISTS load_timeout;
