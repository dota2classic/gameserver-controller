-- 1. Drop the column
ALTER TABLE match_resources
DROP COLUMN IF EXISTS status;

-- 2. Drop the enum type
DROP TYPE IF EXISTS match_status;