DELETE FROM posts WHERE kind <> 'text';

ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_body_check;
ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_kind_check;

ALTER TABLE posts
    DROP COLUMN IF EXISTS kind,
    DROP COLUMN IF EXISTS fen,
    DROP COLUMN IF EXISTS game_id,
    DROP COLUMN IF EXISTS game_pgn,
    DROP COLUMN IF EXISTS game_white_name,
    DROP COLUMN IF EXISTS game_black_name,
    DROP COLUMN IF EXISTS game_white_rating,
    DROP COLUMN IF EXISTS game_black_rating,
    DROP COLUMN IF EXISTS game_result,
    DROP COLUMN IF EXISTS game_end_reason,
    DROP COLUMN IF EXISTS game_mode,
    DROP COLUMN IF EXISTS game_played_at;

ALTER TABLE posts ADD CONSTRAINT posts_body_check
    CHECK (char_length(TRIM(body)) BETWEEN 1 AND 2000);
