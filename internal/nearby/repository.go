package nearby

import (
	"context"
	"errors"
	"log/slog"

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

func (r *Repository) SaveLocation(ctx context.Context, userID int64, latitude float64, longitude float64) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO player_locations (user_id, location, nearby_enabled, updated_at)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, true, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET location = EXCLUDED.location, nearby_enabled = true, updated_at = NOW()`,
		userID, longitude, latitude,
	)
	if err != nil {
		slog.ErrorContext(ctx, "save player location failed", "user_id", userID, "error", err)
		return err
	}
	return nil
}

func (r *Repository) SetNearbyEnabled(ctx context.Context, userID int64, enabled bool) (bool, error) {
	result, err := r.DB.Exec(ctx,
		`UPDATE player_locations SET nearby_enabled = $2 WHERE user_id = $1`,
		userID, enabled,
	)
	if err != nil {
		slog.ErrorContext(ctx, "set nearby enabled failed", "user_id", userID, "error", err)
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *Repository) IsNearbyEnabled(ctx context.Context, userID int64) (bool, error) {
	var enabled bool
	err := r.DB.QueryRow(ctx,
		`SELECT nearby_enabled FROM player_locations WHERE user_id = $1`,
		userID,
	).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "check nearby enabled failed", "user_id", userID, "error", err)
		return false, err
	}
	return enabled, nil
}

func (r *Repository) FindNearbyPlayers(ctx context.Context, userID int64) ([]NearbyPlayer, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT
			u.id,
			u.username,
			u.rating,
			GREATEST(1, ROUND(ST_Distance(other.location, me.location) / 1000))::int
		FROM player_locations me
		JOIN player_locations other ON other.user_id <> me.user_id
		JOIN users u ON u.id = other.user_id
		WHERE me.user_id = $1
			AND me.nearby_enabled = true
			AND other.nearby_enabled = true
			AND ST_DWithin(other.location, me.location, $2)
		ORDER BY ST_Distance(other.location, me.location)
		LIMIT $3`,
		userID, searchRadiusMeters, maxNearbyPlayers,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find nearby players query failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	players := []NearbyPlayer{}
	for rows.Next() {
		var player NearbyPlayer
		err := rows.Scan(&player.UserID, &player.Username, &player.Rating, &player.DistanceKm)
		if err != nil {
			slog.ErrorContext(ctx, "scan nearby player row failed", "user_id", userID, "error", err)
			return nil, err
		}
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "find nearby players rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return players, nil
}
