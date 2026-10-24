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

func (r *Repository) GetDailyLeaderboard(ctx context.Context, limit int, offset int) ([]LeaderboardEntry, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT u.id, u.username, COALESCE(pr.rating, u.start_rating) AS daily_rating
		FROM users u
		LEFT JOIN player_ratings pr ON pr.user_id = u.id AND pr.mode = 'daily'
		ORDER BY daily_rating DESC, u.created_at::date ASC NULLS LAST, LOWER(u.username) ASC
		LIMIT $1 OFFSET $2`,
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
		err := rows.Scan(&entry.UserID, &entry.Username, &entry.Rating)
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
