CREATE TABLE IF NOT EXISTS game_analyses (
    id UUID PRIMARY KEY,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    accuracy_white NUMERIC(5,2),
    accuracy_black NUMERIC(5,2),
    rating_white INT NOT NULL DEFAULT 100,
    rating_black INT NOT NULL DEFAULT 100,
    engine_version VARCHAR(50) NOT NULL DEFAULT 'Stockfish 16',
    depth INT NOT NULL DEFAULT 12,
    brilliant_white INT NOT NULL DEFAULT 0,
    great_white INT NOT NULL DEFAULT 0,
    best_white INT NOT NULL DEFAULT 0,
    good_white INT NOT NULL DEFAULT 0,
    inaccuracies_white INT NOT NULL DEFAULT 0,
    mistakes_white INT NOT NULL DEFAULT 0,
    misses_white INT NOT NULL DEFAULT 0,
    blunders_white INT NOT NULL DEFAULT 0,
    brilliant_black INT NOT NULL DEFAULT 0,
    great_black INT NOT NULL DEFAULT 0,
    best_black INT NOT NULL DEFAULT 0,
    good_black INT NOT NULL DEFAULT 0,
    inaccuracies_black INT NOT NULL DEFAULT 0,
    mistakes_black INT NOT NULL DEFAULT 0,
    misses_black INT NOT NULL DEFAULT 0,
    blunders_black INT NOT NULL DEFAULT 0,
    opening_accuracy_white NUMERIC(5,2),
    opening_accuracy_black NUMERIC(5,2),
    middlegame_accuracy_white NUMERIC(5,2),
    middlegame_accuracy_black NUMERIC(5,2),
    endgame_accuracy_white NUMERIC(5,2),
    endgame_accuracy_black NUMERIC(5,2),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_game_analyses_game_id ON game_analyses(game_id);

CREATE TABLE IF NOT EXISTS game_move_analyses (
    id UUID PRIMARY KEY,
    analysis_id UUID NOT NULL REFERENCES game_analyses(id) ON DELETE CASCADE,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    move_number INT NOT NULL,
    player_color VARCHAR(10) NOT NULL,
    played_move VARCHAR(20) NOT NULL,
    position_fen TEXT NOT NULL,
    eval_before INT,
    eval_after INT,
    best_move VARCHAR(20),
    pv TEXT[],
    centipawn_loss INT NOT NULL DEFAULT 0,
    classification VARCHAR(20) NOT NULL,
    phase VARCHAR(20) NOT NULL DEFAULT 'opening',
    explanation TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_game_move_analyses_analysis_id ON game_move_analyses(analysis_id);
CREATE INDEX idx_game_move_analyses_game_id ON game_move_analyses(game_id);
