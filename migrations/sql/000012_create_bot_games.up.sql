CREATE TABLE IF NOT EXISTS bot_games (
    id UUID PRIMARY KEY,
    player_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    player_color VARCHAR(5) NOT NULL CHECK (player_color IN ('white', 'black')),
    bot_level VARCHAR(20) NOT NULL CHECK (bot_level IN ('easy', 'medium', 'hard', 'expert', 'custom')),
    bot_rating INTEGER NOT NULL CHECK (bot_rating BETWEEN 1000 AND 3000),
    position TEXT NOT NULL,
    moves TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    result VARCHAR(20),
    end_reason VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_bot_games_player_active
    ON bot_games(player_id)
    WHERE status = 'active';
