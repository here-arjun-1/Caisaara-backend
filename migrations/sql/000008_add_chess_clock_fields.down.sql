ALTER TABLE games
DROP COLUMN IF EXISTS turn_started_at,
DROP COLUMN IF EXISTS current_turn,
DROP COLUMN IF EXISTS black_time_ms,
DROP COLUMN IF EXISTS white_time_ms,
DROP COLUMN IF EXISTS increment_ms,
DROP COLUMN IF EXISTS initial_time_ms;
