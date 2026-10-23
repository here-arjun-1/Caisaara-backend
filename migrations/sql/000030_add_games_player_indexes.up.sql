CREATE INDEX IF NOT EXISTS idx_games_white_player_mode ON games(white_player_id, time_control_mode, ended_at) WHERE status = 'finished';
CREATE INDEX IF NOT EXISTS idx_games_black_player_mode ON games(black_player_id, time_control_mode, ended_at) WHERE status = 'finished';
