CREATE TABLE IF NOT EXISTS club_bans (
    club_id UUID NOT NULL REFERENCES clubs(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    banned_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (club_id, user_id)
);
