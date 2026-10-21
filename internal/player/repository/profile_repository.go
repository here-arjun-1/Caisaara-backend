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
			p.avatar_url,
			(SELECT COUNT(*) FROM friend_requests fr
				WHERE (fr.sender_id = u.id OR fr.receiver_id = u.id)
					AND fr.status = 'accepted')
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
		&p.FriendsCount,
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
			COUNT(*) FILTER (WHERE result = 'draw'),
			COALESCE(AVG(EXTRACT(EPOCH FROM (ended_at - started_at))) FILTER (
				WHERE ended_at IS NOT NULL AND started_at IS NOT NULL AND ended_at >= started_at
			), 0)
		FROM games
		WHERE (white_player_id = $1 OR black_player_id = $1)
		  AND ($2 = '' OR time_control_mode = $2)
		  AND rated = true
		  AND status = 'finished'`,
		userID,
		mode,
	).Scan(
		&s.Wins,
		&s.Losses,
		&s.Draws,
		&s.AverageGameTimeSeconds,
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

func (r *ProfileRepository) FindRatingHistory(ctx context.Context, userID int64, mode string, limit int) ([]model.RatingHistoryPoint, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := r.DB.Query(
		ctx,
		`WITH recent_games AS (
			SELECT
				g.id::text AS game_id,
				g.ended_at AS ended_at,
				CASE
					WHEN g.white_player_id = $1 THEN COALESCE(g.white_rating_after, 1200)
					ELSE COALESCE(g.black_rating_after, 1200)
				END AS rating,
				CASE
					WHEN g.white_player_id = $1 THEN (COALESCE(g.white_rating_after, 1200) - COALESCE(g.white_rating_before, 1200))
					ELSE (COALESCE(g.black_rating_after, 1200) - COALESCE(g.black_rating_before, 1200))
				END AS rating_change,
				opp.username AS opponent_username,
				CASE
					WHEN g.result = 'draw' THEN 'draw'
					WHEN (g.white_player_id = $1 AND g.result = 'white_win') OR (g.black_player_id = $1 AND g.result = 'black_win') THEN 'win'
					ELSE 'loss'
				END AS outcome
			FROM games g
			JOIN users opp ON opp.id = CASE
				WHEN g.white_player_id = $1 THEN g.black_player_id
				ELSE g.white_player_id
			END
			WHERE (g.white_player_id = $1 OR g.black_player_id = $1)
			  AND ($2 = '' OR g.time_control_mode = $2)
			  AND g.rated = true
			  AND g.status = 'finished'
			  AND g.ended_at IS NOT NULL
			ORDER BY g.ended_at DESC
			LIMIT $3
		)
		SELECT game_id, ended_at, rating, rating_change, opponent_username, outcome
		FROM recent_games
		ORDER BY ended_at ASC`,
		userID,
		mode,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.RatingHistoryPoint
	for rows.Next() {
		var p model.RatingHistoryPoint
		err := rows.Scan(
			&p.GameID,
			&p.EndedAt,
			&p.Rating,
			&p.RatingChange,
			&p.OpponentUsername,
			&p.Outcome,
		)
		if err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

func (r *ProfileRepository) SearchByUsername(ctx context.Context, prefix string, limit int) ([]model.Profile, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			u.username,
			u.rating,
			p.display_name,
			p.avatar_url
		FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.username LIKE $1 || '%'
		ORDER BY u.username
		LIMIT $2`,
		prefix,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []model.Profile
	for rows.Next() {
		var p model.Profile
		err := rows.Scan(
			&p.Username,
			&p.Rating,
			&p.DisplayName,
			&p.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}

	return profiles, rows.Err()
}
