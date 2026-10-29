ALTER TABLE clubs DROP COLUMN IF EXISTS share_code_expires_at;
ALTER TABLE tournaments DROP COLUMN IF EXISTS invite_code_expires_at;
