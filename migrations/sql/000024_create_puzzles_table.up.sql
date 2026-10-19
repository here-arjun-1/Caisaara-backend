CREATE TABLE IF NOT EXISTS puzzles (
    id VARCHAR(16) PRIMARY KEY,
    fen TEXT NOT NULL,
    moves TEXT NOT NULL,
    rating INT NOT NULL,
    rating_deviation INT NOT NULL DEFAULT 75,
    popularity INT NOT NULL DEFAULT 0,
    nb_plays INT NOT NULL DEFAULT 0,
    themes TEXT[] NOT NULL DEFAULT '{}',
    game_url TEXT,
    opening_tags TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_puzzles_rating ON puzzles(rating);
CREATE INDEX IF NOT EXISTS idx_puzzles_themes ON puzzles USING GIN(themes);
CREATE INDEX IF NOT EXISTS idx_puzzles_popularity ON puzzles(popularity);

CREATE TABLE IF NOT EXISTS user_puzzle_ratings (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    rating INT NOT NULL DEFAULT 1500,
    rating_deviation DOUBLE PRECISION NOT NULL DEFAULT 350,
    rating_volatility DOUBLE PRECISION NOT NULL DEFAULT 0.06,
    puzzles_solved INT NOT NULL DEFAULT 0,
    puzzles_failed INT NOT NULL DEFAULT 0,
    best_rating INT NOT NULL DEFAULT 1500,
    best_rating_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_streak INT NOT NULL DEFAULT 0,
    max_streak INT NOT NULL DEFAULT 0,
    last_played_date DATE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_puzzle_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    puzzle_id VARCHAR(16) NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    solved BOOLEAN NOT NULL,
    user_rating_before INT NOT NULL,
    user_rating_after INT NOT NULL,
    time_taken_ms INT NOT NULL DEFAULT 0,
    attempted_moves TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_puzzle_attempts_user_id ON user_puzzle_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_user_puzzle_attempts_puzzle_id ON user_puzzle_attempts(puzzle_id);
CREATE INDEX IF NOT EXISTS idx_user_puzzle_attempts_user_puzzle ON user_puzzle_attempts(user_id, puzzle_id);

CREATE TABLE IF NOT EXISTS daily_puzzle_sessions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    puzzle_id VARCHAR(16) NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    attempts_left INT NOT NULL DEFAULT 3,
    hint_used BOOLEAN NOT NULL DEFAULT FALSE,
    solution_viewed BOOLEAN NOT NULL DEFAULT FALSE,
    solved BOOLEAN NOT NULL DEFAULT FALSE,
    rating_earned INT NOT NULL DEFAULT 0,
    attempted_moves TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, date)
);

CREATE TABLE IF NOT EXISTS puzzle_rush_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode VARCHAR(16) NOT NULL,
    score INT NOT NULL DEFAULT 0,
    strikes INT NOT NULL DEFAULT 0,
    current_puzzle_id VARCHAR(16) NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    passed_puzzle_ids TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    time_limit_sec INT NOT NULL DEFAULT 180,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_puzzle_rush_sessions_user ON puzzle_rush_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_puzzle_rush_sessions_mode_score ON puzzle_rush_sessions(mode, score DESC);

CREATE TABLE IF NOT EXISTS user_puzzle_rush_stats (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    best_score_3min INT NOT NULL DEFAULT 0,
    best_score_5min INT NOT NULL DEFAULT 0,
    best_score_survival INT NOT NULL DEFAULT 0,
    total_runs INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);



