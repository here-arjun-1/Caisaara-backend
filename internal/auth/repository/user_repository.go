package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) CreateUser(user *model.User) error {

	err := r.DB.QueryRow(
		context.Background(),
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

func (r *UserRepository) FindUserByUsername(username string) (*model.User, error) {

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

func (r *UserRepository) FindUserByEmail(email string) (*model.User, error) {
	var user model.User

	err := r.DB.QueryRow(
		context.Background(),
		`SELECT
			id,
			username,
			email,
			password
		FROM users
		WHERE email = $1`,
		email,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdatePassword(email, hashedPassword string) error {
	_, err := r.DB.Exec(
		context.Background(),
		`UPDATE users SET password = $1 WHERE email = $2`,
		hashedPassword,
		email,
	)
	return err
}

func (r *UserRepository) FindUserByID(id int64) (*model.User, error) {
	var user model.User
	err := r.DB.QueryRow(
		context.Background(),
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
