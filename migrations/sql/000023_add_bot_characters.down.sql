DROP TABLE IF EXISTS bot_progress;

DROP INDEX IF EXISTS idx_bot_games_player_created;

ALTER TABLE bot_games DROP COLUMN IF EXISTS bot_id;

ALTER TABLE bot_games DROP CONSTRAINT IF EXISTS bot_games_bot_rating_check;

ALTER TABLE bot_games
    ADD CONSTRAINT bot_games_bot_rating_check CHECK (bot_rating BETWEEN 1000 AND 3000);
