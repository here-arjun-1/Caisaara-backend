ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS kind VARCHAR(20) NOT NULL DEFAULT 'text',
    ADD COLUMN IF NOT EXISTS fen TEXT,
    ADD COLUMN IF NOT EXISTS game_id UUID,
    ADD COLUMN IF NOT EXISTS game_pgn TEXT,
    ADD COLUMN IF NOT EXISTS game_white_name VARCHAR(50),
    ADD COLUMN IF NOT EXISTS game_black_name VARCHAR(50),
    ADD COLUMN IF NOT EXISTS game_white_rating INTEGER,
    ADD COLUMN IF NOT EXISTS game_black_rating INTEGER,
    ADD COLUMN IF NOT EXISTS game_result VARCHAR(20),
    ADD COLUMN IF NOT EXISTS game_end_reason VARCHAR(50),
    ADD COLUMN IF NOT EXISTS game_mode VARCHAR(20),
    ADD COLUMN IF NOT EXISTS game_played_at TIMESTAMPTZ;

ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_kind_check;
ALTER TABLE posts ADD CONSTRAINT posts_kind_check
    CHECK (kind IN ('text', 'game', 'bot_game', 'position'));

ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_body_check;
ALTER TABLE posts ADD CONSTRAINT posts_body_check CHECK (
    char_length(TRIM(body)) <= 2000
    AND (kind <> 'text' OR char_length(TRIM(body)) >= 1)
);
