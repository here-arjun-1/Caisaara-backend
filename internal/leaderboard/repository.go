package leaderboard

import (
	"context"

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
		WHERE pr.mode = $1 AND pr.rating_deviation <= $2 AND u.deleted_at IS NULL
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
