package game

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository interface {
	CreateGame(ctx context.Context, whitePlayerID int64, blackPlayerID int64, timeControlMinutes int, rated bool) (string, error)
	FindGameByID(ctx context.Context, gameID string) (*Game, error)
	UpdateGameState(ctx context.Context, gameID string, position string, status string, result string) error
	AddMove(ctx context.Context, move *GameMove) error
	GetMoves(ctx context.Context, gameID string) ([]GameMove, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GameRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateGame(
	ctx context.Context,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
	rated bool,
) (string, error) {
	gameID := uuid.New()

	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO games (
			id,
			white_player_id,
			black_player_id,
			time_control_minutes,
			rated,
			position,
			status,
			started_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW())`,
		gameID,
		whitePlayerID,
		blackPlayerID,
		timeControlMinutes,
		rated,
		NewChessGame().FEN(),
	)

	if err != nil {
		slog.ErrorContext(ctx, "create game failed", "error", err)
		return "", err
	}

	return gameID.String(), nil
}

func (r *Repository) FindGameByID(
	ctx context.Context,
	gameID string,
) (*Game, error) {
	var g Game

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			white_player_id,
			black_player_id,
			time_control_minutes,
			rated,
			position,
			status,
			COALESCE(result, ''),
			created_at,
			started_at,
			ended_at
		FROM games
		WHERE id = $1`,
		gameID,
	).Scan(
		&g.ID,
		&g.WhitePlayerID,
		&g.BlackPlayerID,
		&g.TimeControlMinutes,
		&g.Rated,
		&g.Position,
		&g.Status,
		&g.Result,
		&g.CreatedAt,
		&g.StartedAt,
		&g.EndedAt,
	)

	if err != nil {
		slog.ErrorContext(ctx, "find game by id failed", "game_id", gameID, "error", err)
		return nil, err
	}

	return &g, nil
}

func (r *Repository) UpdateGameState(
	ctx context.Context,
	gameID string,
	position string,
	status string,
	result string,
) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE games
		SET position = $1,
			status = $2,
			result = $3,
			ended_at = CASE
				WHEN $2 = 'finished' THEN NOW()
				ELSE ended_at
			END
		WHERE id = $4`,
		position,
		status,
		result,
		gameID,
	)

	if err != nil {
		slog.ErrorContext(ctx, "update game state failed", "game_id", gameID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) AddMove(
	ctx context.Context,
	move *GameMove,
) error {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO game_moves (
			id,
			game_id,
			move_number,
			player_id,
			move,
			position_after,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		move.ID,
		move.GameID,
		move.MoveNumber,
		move.PlayerID,
		move.Move,
		move.PositionAfter,
		move.CreatedAt,
	)

	if err != nil {
		slog.ErrorContext(ctx, "add move failed", "game_id", move.GameID, "move_number", move.MoveNumber, "error", err)
		return err
	}

	return nil
}

func (r *Repository) GetMoves(
	ctx context.Context,
	gameID string,
) ([]GameMove, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			id,
			game_id,
			move_number,
			player_id,
			move,
			position_after,
			created_at
		FROM game_moves
		WHERE game_id = $1
		ORDER BY move_number ASC`,
		gameID,
	)

	if err != nil {
		slog.ErrorContext(ctx, "get moves query failed", "game_id", gameID, "error", err)
		return nil, err
	}

	defer rows.Close()

	var moves []GameMove

	for rows.Next() {
		var move GameMove

		err := rows.Scan(
			&move.ID,
			&move.GameID,
			&move.MoveNumber,
			&move.PlayerID,
			&move.Move,
			&move.PositionAfter,
			&move.CreatedAt,
		)

		if err != nil {
			slog.ErrorContext(ctx, "scan move row failed", "game_id", gameID, "error", err)
			return nil, err
		}

		moves = append(moves, move)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get moves rows iteration error", "game_id", gameID, "error", err)
		return nil, err
	}

	return moves, nil
}

func FormatPlayerUUID(playerID int64) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(strconv.FormatInt(playerID, 10))).String()
}

func NewGameMove(
	gameID string,
	playerID int64,
	moveNumber int,
	move string,
	position string,
) *GameMove {
	return &GameMove{
		ID:            uuid.NewString(),
		GameID:        gameID,
		PlayerID:      FormatPlayerUUID(playerID),
		MoveNumber:    moveNumber,
		Move:          move,
		PositionAfter: position,
	}
}
