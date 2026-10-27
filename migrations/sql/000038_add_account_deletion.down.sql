ALTER TABLE tournament_players
DROP COLUMN IF EXISTS withdrawn;

ALTER TABLE users
DROP COLUMN IF EXISTS deleted_at;
