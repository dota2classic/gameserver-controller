-- Add new enum values safely
ALTER TYPE match_status ADD VALUE IF NOT EXISTS 'pending';
ALTER TYPE match_status ADD VALUE IF NOT EXISTS 'finishing';