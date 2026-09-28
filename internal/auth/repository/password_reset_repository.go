package repository

import (
	"context"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
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

func (r *PasswordResetRepository) SaveOTP(email, otpHash string, expiresAt time.Time) error {
	_, err := r.DB.Exec(
		context.Background(),
		`INSERT INTO password_resets (email, otp_hash, otp_expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE 
		SET otp_hash = EXCLUDED.otp_hash, 
		    otp_expires_at = EXCLUDED.otp_expires_at, 
		    reset_token_hash = NULL, 
		    reset_token_expires_at = NULL,
		    attempts = 0,
		    created_at = NOW()`,
		email,
		otpHash,
		expiresAt,
	)
	return err
}

func (r *PasswordResetRepository) SaveResetToken(email, tokenHash string, expiresAt time.Time) error {
	_, err := r.DB.Exec(
		context.Background(),
		`UPDATE password_resets 
		SET reset_token_hash = $1, reset_token_expires_at = $2, otp_hash = '', otp_expires_at = NOW() 
		WHERE email = $3`,
		tokenHash,
		expiresAt,
		email,
	)
	return err
}

func (r *PasswordResetRepository) FindByEmail(email string) (*model.PasswordReset, error) {
	var pr model.PasswordReset
	err := r.DB.QueryRow(
		context.Background(),
		`SELECT id, email, otp_hash, otp_expires_at, reset_token_hash, reset_token_expires_at, created_at
		FROM password_resets
		WHERE email = $1`,
		email,
	).Scan(
		&pr.ID,
		&pr.Email,
		&pr.OTPHash,
		&pr.OTPExpiresAt,
		&pr.ResetTokenHash,
		&pr.ResetTokenExpiresAt,
		&pr.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *PasswordResetRepository) IncrementAttempts(email string) (int, error) {
	var attempts int
	err := r.DB.QueryRow(
		context.Background(),
		`UPDATE password_resets
		SET attempts = attempts + 1
		WHERE email = $1
		RETURNING attempts`,
		email,
	).Scan(&attempts)
	return attempts, err
}

func (r *PasswordResetRepository) ResetPasswordWithToken(tokenHash, hashedPassword string) error {
	ctx := context.Background()

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var email string
	err = tx.QueryRow(
		ctx,
		`DELETE FROM password_resets
		WHERE reset_token_hash = $1
		AND reset_token_expires_at > NOW()
		RETURNING email`,
		tokenHash,
	).Scan(&email)
	if err != nil {
		return err
	}

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
