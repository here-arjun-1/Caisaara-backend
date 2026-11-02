CREATE TABLE IF NOT EXISTS user_streaks (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_streak INT NOT NULL DEFAULT 0,
    best_streak INT NOT NULL DEFAULT 0,
    last_activity_date DATE,
    streak_status VARCHAR(20) NOT NULL DEFAULT 'active',
    freezes_available INT NOT NULL DEFAULT 2,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Kolkata',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_activity_days (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_date DATE NOT NULL,
    activity_type VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, activity_date)
);

CREATE INDEX IF NOT EXISTS idx_user_activity_days_user_date ON user_activity_days(user_id, activity_date);
