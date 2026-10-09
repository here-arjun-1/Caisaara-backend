package rating

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.DB.Begin(ctx)
}

func (r *Repository) ClaimGame(ctx context.Context, tx pgx.Tx, gameID string) (*Game, error) {
	var g Game

	err := tx.QueryRow(
		ctx,
		`UPDATE games
		SET rating_applied = true
		WHERE id = $1
		  AND status = 'finished'
		  AND rated = true
		  AND rating_applied = false
		  AND time_control_mode IN ('bullet', 'blitz', 'rapid', 'daily')
		  AND result IN ('white_win', 'black_win', 'draw')
		RETURNING white_player_id, black_player_id, time_control_mode, result`,
		gameID,
	).Scan(
		&g.WhitePlayerID,
		&g.BlackPlayerID,
		&g.Mode,
		&g.Result,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &g, nil
}

func (r *Repository) LockRatings(ctx context.Context, tx pgx.Tx, mode string, whiteID int64, blackID int64) (*PlayerRating, *PlayerRating, error) {
	userIDs := []int64{whiteID, blackID}

	_, err := tx.Exec(
		ctx,
		`INSERT INTO player_ratings (user_id, mode, rating)
		SELECT id, $2, rating
		FROM users
		WHERE id = ANY($1)
		ORDER BY id
		ON CONFLICT (user_id, mode) DO NOTHING`,
		userIDs,
		mode,
	)
	if err != nil {
		return nil, nil, err
	}

	rows, err := tx.Query(
		ctx,
		`SELECT
			user_id,
			mode,
			rating,
			rating_deviation,
			rating_volatility,
			games_played,
			wins,
			losses,
			draws,
			best_rating,
			best_rating_at
		FROM player_ratings
		WHERE mode = $2 AND user_id = ANY($1)
		ORDER BY user_id
		FOR UPDATE`,
		userIDs,
		mode,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var white, black *PlayerRating
	for rows.Next() {
		var pr PlayerRating
		err := rows.Scan(
			&pr.UserID,
			&pr.Mode,
			&pr.Rating,
			&pr.RatingDeviation,
			&pr.RatingVolatility,
			&pr.GamesPlayed,
			&pr.Wins,
			&pr.Losses,
			&pr.Draws,
			&pr.BestRating,
			&pr.BestRatingAt,
		)
		if err != nil {
			return nil, nil, err
		}

		if pr.UserID == whiteID {
			white = &pr
		} else {
			black = &pr
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	if white == nil || black == nil {
		return nil, nil, errors.New("player rating row not found")
	}

	return white, black, nil
}

func (r *Repository) SaveRating(ctx context.Context, tx pgx.Tx, pr *PlayerRating) error {
	_, err := tx.Exec(
		ctx,
		`UPDATE player_ratings
		SET rating = $3,
			rating_deviation = $4,
			rating_volatility = $5,
			games_played = $6,
			wins = $7,
			losses = $8,
			draws = $9,
			best_rating = $10,
			best_rating_at = $11,
			updated_at = NOW()
		WHERE user_id = $1 AND mode = $2`,
		pr.UserID,
		pr.Mode,
		pr.Rating,
		pr.RatingDeviation,
		pr.RatingVolatility,
		pr.GamesPlayed,
		pr.Wins,
		pr.Losses,
		pr.Draws,
		pr.BestRating,
		pr.BestRatingAt,
	)
	return err
}
