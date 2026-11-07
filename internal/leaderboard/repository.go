package leaderboard

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

func (r *Repository) GetLeaderboard(ctx context.Context, mode string, maxRD float64, limit int, offset int) ([]LeaderboardEntry, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			RANK() OVER (ORDER BY pr.rating DESC) AS rank,
			u.id,
			u.username,
			pr.rating
		FROM player_ratings pr
		JOIN users u ON u.id = pr.user_id
		WHERE pr.mode = $1 AND pr.rating_deviation <= $2 AND u.deleted_at IS NULL AND u.role = 'user'
		ORDER BY pr.rating DESC, LOWER(u.username) ASC, u.id ASC
		LIMIT $3 OFFSET $4`,
		mode,
		maxRD,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []LeaderboardEntry{}
	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(&entry.Rank, &entry.UserID, &entry.Username, &entry.Rating)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *Repository) GetLocationStatus(ctx context.Context, userID int64, maxAgeDays int) (bool, bool, bool, error) {
	var nearbyEnabled bool
	var locationFresh bool
	err := r.DB.QueryRow(
		ctx,
		`SELECT nearby_enabled, updated_at > NOW() - make_interval(days => $2)
		FROM player_locations WHERE user_id = $1`,
		userID,
		maxAgeDays,
	).Scan(&nearbyEnabled, &locationFresh)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, false, nil
	}
	if err != nil {
		return false, false, false, err
	}

	return true, nearbyEnabled, locationFresh, nil
}

func (r *Repository) GetLocalLeaderboard(ctx context.Context, userID int64, mode string, radiusMeters int, maxAgeDays int, maxRD float64, limit int, offset int) ([]LocalLeaderboardEntry, error) {
	rows, err := r.DB.Query(
		ctx,
		`WITH me AS (
			SELECT location FROM player_locations WHERE user_id = $1
		)
		SELECT
			RANK() OVER (ORDER BY pr.rating DESC) AS rank,
			u.id,
			u.username,
			pr.rating,
			CASE
				WHEN u.id = $1 THEN 0
				ELSE GREATEST(1, ROUND(ST_Distance(pl.location, me.location) / 1000))::int
			END
		FROM me
		JOIN player_locations pl ON ST_DWithin(pl.location, me.location, $3)
		JOIN player_ratings pr ON pr.user_id = pl.user_id AND pr.mode = $2
		JOIN users u ON u.id = pl.user_id
		WHERE pl.nearby_enabled = true
			AND pl.updated_at > NOW() - make_interval(days => $4)
			AND pr.rating_deviation <= $5
			AND u.deleted_at IS NULL
			AND u.role = 'user'
			AND (u.banned_until IS NULL OR u.banned_until <= NOW())
			AND NOT EXISTS (
				SELECT 1 FROM user_blocks ub
				WHERE (ub.blocker_id = $1 AND ub.blocked_id = u.id)
					OR (ub.blocker_id = u.id AND ub.blocked_id = $1)
			)
		ORDER BY pr.rating DESC, LOWER(u.username) ASC, u.id ASC
		LIMIT $6 OFFSET $7`,
		userID,
		mode,
		radiusMeters,
		maxAgeDays,
		maxRD,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []LocalLeaderboardEntry{}
	for rows.Next() {
		var entry LocalLeaderboardEntry
		err := rows.Scan(&entry.Rank, &entry.UserID, &entry.Username, &entry.Rating, &entry.DistanceKm)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
