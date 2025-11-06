-- Recreate the enum type without the extra values
DO $$
BEGIN
    -- 1. Rename the existing type
ALTER TYPE match_status RENAME TO match_status_old;

-- 2. Recreate the original type (without 'pending' and 'finishing')
CREATE TYPE match_status AS ENUM ('launching', 'running', 'failed', 'done');

-- 3. Update the table column to use the new type
ALTER TABLE match_resources
ALTER COLUMN status TYPE match_status
        USING status::text::match_status;

    -- 4. Drop the old type
DROP TYPE match_status_old;
END$$;