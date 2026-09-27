package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func (r *UserRepository) FindUserByID(d int64) (any, error) {
	panic("unimplemented")
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
