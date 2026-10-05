package game

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository interface {
	CreateGame(ctx context.Context, whitePlayerID int64, blackPlayerID int64, timeControlMinutes int) (string, error)
	FindGameByID(ctx context.Context, gameID string) (*Game, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateGame(
	ctx context.Context,
	whitePlayerID int64,
	blackPlayerID int64,
	timeControlMinutes int,
) (string, error) {

	gameID := uuid.New()

	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO games (
			id,
			white_player_id,
			black_player_id,
			time_control_minutes,
			status,
			started_at
		)
		VALUES ($1, $2, $3, $4, 'active', NOW())`,
		gameID,
		whitePlayerID,
		blackPlayerID,
		timeControlMinutes,
	)

	if err != nil {
		slog.ErrorContext(ctx, "create game failed", "error", err)
		return "", err
	}

	return gameID.String(), nil
}

func (r *Repository) FindGameByID(ctx context.Context, gameID string) (*Game, error) {
	var g Game
	err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			white_player_id,
			black_player_id,
			time_control_minutes,
			status,
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
		&g.Status,
		&g.CreatedAt,
		&g.StartedAt,
		&g.EndedAt,
	)

	if err != nil {
		return nil, err
	}

	return &g, nil
}
