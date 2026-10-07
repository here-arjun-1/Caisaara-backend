CREATE TABLE IF NOT EXISTS tournaments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    format VARCHAR(20) NOT NULL DEFAULT 'swiss' CHECK (format IN ('swiss', 'knockout', 'round_robin')),
    time_control VARCHAR(50) NOT NULL,
    max_players INT NOT NULL CHECK (max_players > 0),
    visibility VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
    invite_code VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'registration' CHECK (status IN ('registration', 'upcoming', 'ongoing', 'completed', 'cancelled')),
    total_rounds INT NOT NULL DEFAULT 0,
    current_round INT NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tournament_players (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score NUMERIC(4, 1) NOT NULL DEFAULT 0.0,
    wins INT NOT NULL DEFAULT 0,
    draws INT NOT NULL DEFAULT 0,
    losses INT NOT NULL DEFAULT 0,
    games_played INT NOT NULL DEFAULT 0,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tournament_player UNIQUE (tournament_id, user_id)
);

CREATE TABLE IF NOT EXISTS tournament_rounds (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round_number INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ongoing', 'completed')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    CONSTRAINT uq_tournament_round UNIQUE (tournament_id, round_number)
);

CREATE TABLE IF NOT EXISTS tournament_pairings (
    id BIGSERIAL PRIMARY KEY,
    tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round_id BIGINT NOT NULL REFERENCES tournament_rounds(id) ON DELETE CASCADE,
    white_player_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    black_player_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    game_id UUID REFERENCES games(id) ON DELETE SET NULL,
    result VARCHAR(20) CHECK (result IS NULL OR result IN ('white_win', 'black_win', 'draw', 'bye')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ongoing', 'completed')),
    is_bye BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tournaments_status ON tournaments(status);
CREATE INDEX IF NOT EXISTS idx_tournament_players_tournament_id ON tournament_players(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_rounds_tournament_id ON tournament_rounds(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_pairings_round_id ON tournament_pairings(round_id);
CREATE INDEX IF NOT EXISTS idx_tournament_pairings_tournament_id ON tournament_pairings(tournament_id);
