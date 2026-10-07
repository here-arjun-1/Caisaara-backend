package bot

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository interface {
	CreateGame(ctx context.Context, game *Game) error
	FindGameByID(ctx context.Context, gameID string) (*Game, error)
	CountActiveGames(ctx context.Context, playerID int64) (int, error)
	UpdateGame(ctx context.Context, game *Game, previousMoves []string) error
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GameRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateGame(ctx context.Context, game *Game) error {
	game.ID = uuid.New().String()

	err := r.DB.QueryRow(
		ctx,
		`INSERT INTO bot_games (
			id,
			player_id,
			player_color,
			bot_level,
			bot_rating,
			position,
			moves,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`,
		game.ID,
		game.PlayerID,
		game.PlayerColor,
		game.BotLevel,
		game.BotRating,
		game.Position,
		strings.Join(game.Moves, " "),
		game.Status,
	).Scan(&game.CreatedAt)

	if err != nil {
		slog.ErrorContext(ctx, "create bot game failed", "player_id", game.PlayerID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) FindGameByID(ctx context.Context, gameID string) (*Game, error) {
	if _, err := uuid.Parse(gameID); err != nil {
		return nil, ErrGameNotFound
	}

	row := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			player_id,
			player_color,
			bot_level,
			bot_rating,
			position,
			moves,
			status,
			COALESCE(result, ''),
			COALESCE(end_reason, ''),
			created_at,
			ended_at
		FROM bot_games
		WHERE id = $1`,
		gameID,
	)

	return scanGame(ctx, row)
}

func (r *Repository) CountActiveGames(ctx context.Context, playerID int64) (int, error) {
	var count int

	err := r.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		FROM bot_games
		WHERE player_id = $1 AND status = 'active'`,
		playerID,
	).Scan(&count)
	if err != nil {
		slog.ErrorContext(ctx, "count active bot games failed", "player_id", playerID, "error", err)
		return 0, err
	}

	return count, nil
}

func (r *Repository) UpdateGame(ctx context.Context, game *Game, previousMoves []string) error {
	tag, err := r.DB.Exec(
		ctx,
		`UPDATE bot_games
		SET
			position = $2,
			moves = $3,
			status = $4,
			result = NULLIF($5, ''),
			end_reason = NULLIF($6, ''),
			ended_at = $7
		WHERE id = $1
		  AND moves = $8
		  AND status = 'active'`,
		game.ID,
		game.Position,
		strings.Join(game.Moves, " "),
		game.Status,
		game.Result,
		game.EndReason,
		game.EndedAt,
		strings.Join(previousMoves, " "),
	)
	if err != nil {
		slog.ErrorContext(ctx, "update bot game failed", "game_id", game.ID, "error", err)
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrGameChanged
	}

	return nil
}

func scanGame(ctx context.Context, row pgx.Row) (*Game, error) {
	var g Game
	var moves string

	err := row.Scan(
		&g.ID,
		&g.PlayerID,
		&g.PlayerColor,
		&g.BotLevel,
		&g.BotRating,
		&g.Position,
		&moves,
		&g.Status,
		&g.Result,
		&g.EndReason,
		&g.CreatedAt,
		&g.EndedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGameNotFound
		}
		slog.ErrorContext(ctx, "scan bot game failed", "error", err)
		return nil, err
	}

	g.Moves = strings.Fields(moves)

	return &g, nil
}
