package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordResetRepository struct {
	DB *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{
		DB: db,
	}
}

func (r *PasswordResetRepository) ResetPassword(email, hashedPassword string) error {
	ctx := context.Background()

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID int64
	err = tx.QueryRow(
		ctx,
		`UPDATE users SET password = $1 WHERE email = $2 RETURNING id`,
		hashedPassword,
		email,
	).Scan(&userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = $1
		AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
