package repository

import (
	"context"
	"errors"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/player/model"
	"github.com/jackc/pgx/v5"
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
			u.start_rating,
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
		&p.StartRating,
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
			u.start_rating,
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
		&p.StartRating,
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
			wins,
			losses,
			draws,
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
			&mr.Wins,
			&mr.Losses,
			&mr.Draws,
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
			COUNT(*) FILTER (WHERE white_player_id = $1 AND result = 'white_win'),
			COUNT(*) FILTER (WHERE white_player_id = $1 AND result = 'black_win'),
			COUNT(*) FILTER (WHERE white_player_id = $1 AND result = 'draw'),
			COUNT(*) FILTER (WHERE black_player_id = $1 AND result = 'black_win'),
			COUNT(*) FILTER (WHERE black_player_id = $1 AND result = 'white_win'),
			COUNT(*) FILTER (WHERE black_player_id = $1 AND result = 'draw')
		FROM games
		WHERE (white_player_id = $1 OR black_player_id = $1)
		  AND time_control_mode = $2
		  AND rated = true
		  AND status = 'finished'`,
		userID,
		mode,
	).Scan(
		&s.WhiteWins,
		&s.WhiteLosses,
		&s.WhiteDraws,
		&s.BlackWins,
		&s.BlackLosses,
		&s.BlackDraws,
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

func (r *ProfileRepository) FindRatingHistory(ctx context.Context, userID int64, mode string, since time.Time) ([]model.RatingPoint, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT DISTINCT ON (game_date) game_date, rating
		FROM (
			SELECT
				to_char(ended_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS game_date,
				CASE WHEN white_player_id = $1 THEN white_rating_after ELSE black_rating_after END AS rating,
				ended_at
			FROM games
			WHERE (white_player_id = $1 OR black_player_id = $1)
			  AND time_control_mode = $2
			  AND rated = true
			  AND status = 'finished'
			  AND ended_at >= $3
		) t
		WHERE rating IS NOT NULL
		ORDER BY game_date, ended_at DESC`,
		userID,
		mode,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []model.RatingPoint
	for rows.Next() {
		var p model.RatingPoint
		if err := rows.Scan(&p.Date, &p.Rating); err != nil {
			return nil, err
		}
		points = append(points, p)
	}

	return points, rows.Err()
}

func (r *ProfileRepository) FindStartRating(ctx context.Context, userID int64, mode string, since time.Time) (*int, error) {
	var rating int

	err := r.DB.QueryRow(
		ctx,
		`SELECT rating
		FROM (
			SELECT
				CASE WHEN white_player_id = $1 THEN white_rating_before ELSE black_rating_before END AS rating,
				ended_at
			FROM games
			WHERE (white_player_id = $1 OR black_player_id = $1)
			  AND time_control_mode = $2
			  AND rated = true
			  AND status = 'finished'
			  AND ended_at >= $3
		) t
		WHERE rating IS NOT NULL
		ORDER BY ended_at
		LIMIT 1`,
		userID,
		mode,
		since,
	).Scan(&rating)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &rating, nil
}

func (r *ProfileRepository) FindLowestRating(ctx context.Context, userID int64, mode string) (*model.LowestRating, error) {
	var lowest model.LowestRating

	err := r.DB.QueryRow(
		ctx,
		`SELECT rating, ended_at
		FROM (
			SELECT
				CASE WHEN white_player_id = $1 THEN white_rating_after ELSE black_rating_after END AS rating,
				ended_at
			FROM games
			WHERE (white_player_id = $1 OR black_player_id = $1)
			  AND time_control_mode = $2
			  AND rated = true
			  AND status = 'finished'
		) t
		WHERE rating IS NOT NULL
		ORDER BY rating, ended_at
		LIMIT 1`,
		userID,
		mode,
	).Scan(&lowest.Rating, &lowest.At)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &lowest, nil
}

func (r *ProfileRepository) FindGameResults(ctx context.Context, userID int64, mode string) ([]model.GameResult, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			CASE WHEN white_player_id = $1 THEN 'white' ELSE 'black' END,
			COALESCE(result, '')
		FROM games
		WHERE (white_player_id = $1 OR black_player_id = $1)
		  AND time_control_mode = $2
		  AND rated = true
		  AND status = 'finished'
		ORDER BY ended_at ASC NULLS FIRST`,
		userID,
		mode,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.GameResult
	for rows.Next() {
		var g model.GameResult
		if err := rows.Scan(&g.Color, &g.Result); err != nil {
			return nil, err
		}
		results = append(results, g)
	}

	return results, rows.Err()
}

func (r *ProfileRepository) SearchByUsername(ctx context.Context, prefix string, limit int) ([]model.Profile, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			u.id,
			u.username,
			u.rating,
			p.display_name,
			p.avatar_url
		FROM users u
		LEFT JOIN player_profiles p ON p.user_id = u.id
		WHERE u.username LIKE $1 || '%' OR CAST(u.id AS TEXT) = $1
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
			&p.UserID,
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
