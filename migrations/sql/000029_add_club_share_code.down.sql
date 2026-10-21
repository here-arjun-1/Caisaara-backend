DROP INDEX IF EXISTS idx_clubs_share_code;

ALTER TABLE clubs DROP COLUMN IF EXISTS share_code;
