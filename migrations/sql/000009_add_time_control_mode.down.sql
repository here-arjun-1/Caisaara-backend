ALTER TABLE games
DROP COLUMN IF EXISTS daily_move_time_ms,
DROP COLUMN IF EXISTS time_control_mode;
