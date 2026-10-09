package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/player/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProfileRepository struct {
	DB *pgxpool.Pool
}

func NewProfileRepository(db *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{
		DB: db,
	}
}

func (r *ProfileRepository) FindByUserID(ctx context.Context, userID int64) (*model.Profile, error) {
	var p model.Profile

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			u.id,
			u.username,
			u.email,
			u.rating,
			u.rating_deviation,
			u.skill_level,
			u.created_at,
			p.display_name,
			p.country,
			p.bio,
			p.avatar_url
		FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.id = $1`,
		userID,
	).Scan(
		&p.UserID,
		&p.Username,
		&p.Email,
		&p.Rating,
		&p.RatingDeviation,
		&p.SkillLevel,
		&p.CreatedAt,
		&p.DisplayName,
		&p.Country,
		&p.Bio,
		&p.AvatarURL,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *ProfileRepository) FindByUsername(ctx context.Context, username string) (*model.Profile, error) {
	var p model.Profile

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			u.id,
			u.username,
			u.email,
			u.rating,
			u.rating_deviation,
			u.skill_level,
			u.created_at,
			p.display_name,
			p.country,
			p.bio,
			p.avatar_url
		FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.username = $1`,
		username,
	).Scan(
		&p.UserID,
		&p.Username,
		&p.Email,
		&p.Rating,
		&p.RatingDeviation,
		&p.SkillLevel,
		&p.CreatedAt,
		&p.DisplayName,
		&p.Country,
		&p.Bio,
		&p.AvatarURL,
	)

	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *ProfileRepository) SaveProfile(ctx context.Context, p *model.Profile) error {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO player_profiles (user_id, display_name, country, bio, avatar_url)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			country      = EXCLUDED.country,
			bio          = EXCLUDED.bio,
			avatar_url   = EXCLUDED.avatar_url,
			updated_at   = now()`,
		p.UserID,
		p.DisplayName,
		p.Country,
		p.Bio,
		p.AvatarURL,
	)
	return err
}

func (r *ProfileRepository) FindModeRatings(ctx context.Context, userID int64) ([]model.ModeRating, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			mode,
			rating,
			rating_deviation,
			games_played,
			best_rating,
			best_rating_at
		FROM player_ratings
		WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []model.ModeRating
	for rows.Next() {
		var mr model.ModeRating
		err := rows.Scan(
			&mr.Mode,
			&mr.Rating,
			&mr.RatingDeviation,
			&mr.GamesPlayed,
			&mr.BestRating,
			&mr.BestRatingAt,
		)
		if err != nil {
			return nil, err
		}
		ratings = append(ratings, mr)
	}

	return ratings, rows.Err()
}

func (r *ProfileRepository) FindModeGameStats(ctx context.Context, userID int64, mode string) (*model.ModeGameStats, error) {
	var s model.ModeGameStats

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			COUNT(*) FILTER (
				WHERE (white_player_id = $1 AND result = 'white_win')
				   OR (black_player_id = $1 AND result = 'black_win')
			),
			COUNT(*) FILTER (
				WHERE (white_player_id = $1 AND result = 'black_win')
				   OR (black_player_id = $1 AND result = 'white_win')
			),
			COUNT(*) FILTER (WHERE result = 'draw')
		FROM games
		WHERE (white_player_id = $1 OR black_player_id = $1)
		  AND time_control_mode = $2
		  AND rated = true
		  AND status = 'finished'`,
		userID,
		mode,
	).Scan(
		&s.Wins,
		&s.Losses,
		&s.Draws,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *ProfileRepository) FindRecentGamesByMode(ctx context.Context, userID int64, mode string, limit int) ([]model.RecentGame, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			g.id::text,
			opp.username,
			CASE WHEN g.white_player_id = $1 THEN 'white' ELSE 'black' END,
			COALESCE(g.result, ''),
			g.end_reason,
			g.ended_at
		FROM games g
		JOIN users opp ON opp.id = CASE
			WHEN g.white_player_id = $1 THEN g.black_player_id
			ELSE g.white_player_id
		END
		WHERE (g.white_player_id = $1 OR g.black_player_id = $1)
		  AND g.time_control_mode = $2
		  AND g.rated = true
		  AND g.status = 'finished'
		ORDER BY g.ended_at DESC NULLS LAST
		LIMIT $3`,
		userID,
		mode,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []model.RecentGame
	for rows.Next() {
		var g model.RecentGame
		err := rows.Scan(
			&g.GameID,
			&g.OpponentUsername,
			&g.Color,
			&g.Result,
			&g.EndReason,
			&g.EndedAt,
		)
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}

	return games, rows.Err()
}
