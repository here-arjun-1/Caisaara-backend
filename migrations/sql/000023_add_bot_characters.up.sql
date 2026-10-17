ALTER TABLE bot_games DROP CONSTRAINT IF EXISTS bot_games_bot_rating_check;

ALTER TABLE bot_games
    ADD CONSTRAINT bot_games_bot_rating_check CHECK (bot_rating BETWEEN 400 AND 3000);

ALTER TABLE bot_games ADD COLUMN IF NOT EXISTS bot_id VARCHAR(30);

CREATE INDEX IF NOT EXISTS idx_bot_games_player_created
    ON bot_games(player_id, created_at DESC);

CREATE TABLE IF NOT EXISTS bot_progress (
    player_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bot_id VARCHAR(30) NOT NULL,
    game_id UUID NOT NULL REFERENCES bot_games(id) ON DELETE CASCADE,
    won_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (player_id, bot_id)
);
