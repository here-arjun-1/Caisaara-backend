ALTER TABLE clubs ADD COLUMN IF NOT EXISTS share_code VARCHAR(8);

UPDATE clubs
SET share_code = upper(substr(md5(random()::text || id::text), 1, 8))
WHERE share_code IS NULL;

ALTER TABLE clubs ALTER COLUMN share_code SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_clubs_share_code ON clubs(share_code);
