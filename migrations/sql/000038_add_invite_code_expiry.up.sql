ALTER TABLE clubs ADD COLUMN IF NOT EXISTS share_code_expires_at TIMESTAMPTZ;
UPDATE clubs SET share_code_expires_at = created_at + INTERVAL '7 days' WHERE share_code_expires_at IS NULL;

ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS invite_code_expires_at TIMESTAMPTZ;
UPDATE tournaments SET invite_code_expires_at = created_at + INTERVAL '7 days' WHERE invite_code_expires_at IS NULL;
