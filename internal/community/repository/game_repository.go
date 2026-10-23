package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/community/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GameRepository struct {
	DB *pgxpool.Pool
}

func NewGameRepository(db *pgxpool.Pool) *GameRepository {
	return &GameRepository{
		DB: db,
	}
}

func (r *GameRepository) FindGame(ctx context.Context, gameID string) (*model.GameForPost, error) {
	var g model.GameForPost

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			g.white_player_id,
			g.black_player_id,
			COALESCE(uw.username, ''),
			COALESCE(ub.username, ''),
			g.white_rating_before,
			g.black_rating_before,
			g.status,
			COALESCE(g.result, ''),
			COALESCE(g.end_reason, ''),
			g.time_control_mode,
			g.position,
			g.ended_at
		FROM games g
		LEFT JOIN users uw ON uw.id = g.white_player_id
		LEFT JOIN users ub ON ub.id = g.black_player_id
		WHERE g.id = $1`,
		gameID,
	).Scan(
		&g.WhitePlayerID,
		&g.BlackPlayerID,
		&g.WhiteName,
		&g.BlackName,
		&g.WhiteRating,
		&g.BlackRating,
		&g.Status,
		&g.Result,
		&g.EndReason,
		&g.Mode,
		&g.Position,
		&g.EndedAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := r.DB.Query(
		ctx,
		`SELECT move FROM game_moves WHERE game_id = $1 ORDER BY move_number`,
		gameID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	g.Moves = []string{}
	for rows.Next() {
		var move string
		if err := rows.Scan(&move); err != nil {
			return nil, err
		}
		g.Moves = append(g.Moves, move)
	}

	return &g, rows.Err()
}

func (r *GameRepository) FindBotGame(ctx context.Context, gameID string) (*model.BotGameForPost, error) {
	var g model.BotGameForPost

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			b.player_id,
			b.player_color,
			u.username,
			COALESCE(b.bot_id, ''),
			b.bot_rating,
			b.status,
			COALESCE(b.result, ''),
			COALESCE(b.end_reason, ''),
			b.position,
			b.moves,
			b.ended_at
		FROM bot_games b
		JOIN users u ON u.id = b.player_id
		WHERE b.id = $1`,
		gameID,
	).Scan(
		&g.PlayerID,
		&g.PlayerColor,
		&g.Username,
		&g.BotID,
		&g.BotRating,
		&g.Status,
		&g.Result,
		&g.EndReason,
		&g.Position,
		&g.Moves,
		&g.EndedAt,
	)
	if err != nil {
		return nil, err
	}

	return &g, nil
}
