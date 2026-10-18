CREATE INDEX IF NOT EXISTS idx_games_active_status ON games(status) WHERE status = 'active';
