-- Drop image column
ALTER TABLE gameserver_settings
DROP COLUMN IF EXISTS cpu_affinity;