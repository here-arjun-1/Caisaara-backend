ALTER TABLE games
ADD COLUMN position TEXT NOT NULL DEFAULT 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1';

ALTER TABLE games
ADD COLUMN result VARCHAR(20);

CREATE TABLE game_moves (
    id UUID PRIMARY KEY,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    move_number INT NOT NULL,
    player_id UUID NOT NULL,
    move VARCHAR(20) NOT NULL,
    position_after TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_game_moves_game_id
ON game_moves(game_id);