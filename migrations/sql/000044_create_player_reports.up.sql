CREATE TABLE IF NOT EXISTS player_reports (
    id BIGSERIAL PRIMARY KEY,
    reporter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reported_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason VARCHAR(20) NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (reporter_id <> reported_id),
    CHECK (reason IN ('cheating', 'abuse', 'other'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_reports_one_pending
    ON player_reports (reporter_id, reported_id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_player_reports_status_created
    ON player_reports (status, created_at DESC);
