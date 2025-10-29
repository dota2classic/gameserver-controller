-- 1. Create the enum type
CREATE TYPE match_status AS ENUM ('launching', 'running', 'failed', 'done');

-- 2. Add the column to your table
ALTER TABLE match_resources
ADD COLUMN status match_status NOT NULL DEFAULT 'launching';