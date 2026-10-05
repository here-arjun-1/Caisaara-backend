CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY,
    white_player_id BIGINT NOT NULL REFERENCES users(id),
    black_player_id BIGINT NOT NULL REFERENCES users(id),
    time_control_minutes INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ
);
