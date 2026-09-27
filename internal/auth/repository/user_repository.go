package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	DB *pgx.Conn
}

func (r *UserRepository) FindUserByID(d int64) (any, error) {
	panic("unimplemented")
}

func NewUserRepository(db *pgx.Conn) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) CreateUser(
	user *model.User,
) error {

	err := r.DB.QueryRow(
		context.Background(),
		`INSERT INTO users
		(username, email, password, email_verified,
		 email_verification_token, email_verification_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		user.Username,
		user.Email,
		user.Password,
		user.EmailVerified,
		user.EmailVerificationToken,
		user.EmailVerificationExpiresAt,
	).Scan(&user.ID)

	return err
}

func (r *UserRepository) FindUserByUsername(
	username string,
) (*model.User, error) {

	var user model.User

	err := r.DB.QueryRow(
		context.Background(),
		`SELECT
			id,
			username,
			password
		FROM users
		WHERE username = $1`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) VerifyEmail(
	verificationToken string,
) error {

	result, err := r.DB.Exec(
		context.Background(),
		`UPDATE users
		SET
			email_verified = true,
			email_verification_token = NULL,
			email_verification_expires_at = NULL
		WHERE email_verification_token = $1
		AND email_verification_expires_at > NOW()`,
		verificationToken,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
