package tilt

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository interface {
	GetRecentRatedGames(ctx context.Context, playerID int64, limit int) ([]RecentGame, error)
	GetMoveTimes(ctx context.Context, gameID string) ([]time.Time, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) GameRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) GetRecentRatedGames(
	ctx context.Context,
	playerID int64,
	limit int,
) ([]RecentGame, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			id,
			white_player_id,
			black_player_id,
			COALESCE(time_control_mode, 'rapid'),
			COALESCE(result, ''),
			ended_at
		FROM games
		WHERE (white_player_id = $1 OR black_player_id = $1)
			AND rated = true
			AND status = 'finished'
			AND ended_at IS NOT NULL
		ORDER BY ended_at DESC
		LIMIT $2`,
		playerID,
		limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get recent rated games query failed", "player_id", playerID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var games []RecentGame
	for rows.Next() {
		var g RecentGame
		err := rows.Scan(
			&g.ID,
			&g.WhitePlayerID,
			&g.BlackPlayerID,
			&g.TimeControlMode,
			&g.Result,
			&g.EndedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan recent game row failed", "player_id", playerID, "error", err)
			return nil, err
		}
		games = append(games, g)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get recent rated games rows iteration error", "player_id", playerID, "error", err)
		return nil, err
	}

	return games, nil
}

func (r *Repository) GetMoveTimes(
	ctx context.Context,
	gameID string,
) ([]time.Time, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT created_at FROM game_moves WHERE game_id = $1 ORDER BY move_number ASC`,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get move times query failed", "game_id", gameID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var times []time.Time
	for rows.Next() {
		var createdAt time.Time
		if err := rows.Scan(&createdAt); err != nil {
			slog.ErrorContext(ctx, "scan move time row failed", "game_id", gameID, "error", err)
			return nil, err
		}
		times = append(times, createdAt)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "get move times rows iteration error", "game_id", gameID, "error", err)
		return nil, err
	}

	return times, nil
}
