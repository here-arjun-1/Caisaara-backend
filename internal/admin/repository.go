package admin

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

func (r *Repository) SearchUsers(ctx context.Context, search string, limit int, offset int) ([]User, error) {
	pattern := "%" + search + "%"

	rows, err := r.DB.Query(ctx,
		`SELECT
			id,
			username,
			COALESCE(email, ''),
			role,
			rating,
			COALESCE(banned_until > NOW(), false),
			ban_reason,
			created_at
		FROM users
		WHERE deleted_at IS NULL
			AND (username ILIKE $1 OR COALESCE(email, '') ILIKE $1)
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`,
		pattern,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&user.Rating,
			&user.IsBanned,
			&user.BanReason,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
