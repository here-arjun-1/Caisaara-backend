package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func (r *UserRepository) FindUserByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			username,
			email
		FROM users
		WHERE id = $1`,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *model.User) error {

	err := r.DB.QueryRow(
		ctx,
		`INSERT INTO users
		(username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id`,
		user.Username,
		user.Email,
		user.Password,
	).Scan(&user.ID)

	return err
}

func (r *UserRepository) CreateUserTx(ctx context.Context, tx pgx.Tx, user *model.User) error {
	err := tx.QueryRow(
		ctx,
		`INSERT INTO users
		(username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id`,
		user.Username,
		user.Email,
		user.Password,
	).Scan(&user.ID)

	return err
}

func (r *UserRepository) FindUserByUsername(ctx context.Context, username string) (*model.User, error) {

	var user model.User

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			username,
			password,
			skill_level
		FROM users
		WHERE username = $1`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.SkillLevel,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User

	err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			username,
			email,
			password,
			skill_level
		FROM users
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.SkillLevel,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, email, hashedPassword string) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE users SET password = $1 WHERE email = $2`,
		hashedPassword,
		email,
	)
	return err
}

func (r *UserRepository) SetInitialRating(ctx context.Context, userID int64, level string, rating int) (bool, error) {
	tag, err := r.DB.Exec(
		ctx,
		`UPDATE users
		SET skill_level = $1, rating = $2
		WHERE id = $3 AND skill_level IS NULL`,
		level,
		rating,
		userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
