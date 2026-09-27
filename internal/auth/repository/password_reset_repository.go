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

func (r *PasswordResetRepository) FindByResetTokenHash(tokenHash string) (*model.PasswordReset, error) {
	var pr model.PasswordReset
	err := r.DB.QueryRow(
		context.Background(),
		`SELECT id, email, otp_hash, otp_expires_at, reset_token_hash, reset_token_expires_at, created_at
		FROM password_resets
		WHERE reset_token_hash = $1`,
		tokenHash,
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

func (r *PasswordResetRepository) DeletePasswordReset(email string) error {
	_, err := r.DB.Exec(
		context.Background(),
		`DELETE FROM password_resets WHERE email = $1`,
		email,
	)
	return err
}
