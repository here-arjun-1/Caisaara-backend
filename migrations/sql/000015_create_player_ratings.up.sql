CREATE TABLE IF NOT EXISTS player_ratings (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode VARCHAR(20) NOT NULL,
    rating INTEGER NOT NULL DEFAULT 1200,
    rating_deviation DOUBLE PRECISION NOT NULL DEFAULT 350,
    rating_volatility DOUBLE PRECISION NOT NULL DEFAULT 0.06,
    games_played INTEGER NOT NULL DEFAULT 0,
    best_rating INTEGER,
    best_rating_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, mode),
    CONSTRAINT player_ratings_mode_check CHECK (mode IN ('bullet', 'blitz', 'rapid', 'daily'))
);

