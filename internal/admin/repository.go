package admin

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

func (r *Repository) GetUserRole(ctx context.Context, userID int64) (string, error) {
	var role string
	err := r.DB.QueryRow(ctx,
		`SELECT role FROM users WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

func (r *Repository) BanUser(ctx context.Context, adminID int64, userID int64, days int, reason string, details string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx,
		`UPDATE users
		SET banned_until = CASE WHEN $2 = 0 THEN 'infinity'::timestamptz ELSE NOW() + make_interval(days => $2) END,
			ban_reason = $3,
			password_version = COALESCE(password_version, 1) + 1
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
		days,
		reason,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	_, err = tx.Exec(ctx,
		`UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO admin_actions (admin_id, action, target_type, target_id, details)
		VALUES ($1, 'ban_user', 'user', $2, $3)`,
		adminID,
		userID,
		details,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
