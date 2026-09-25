package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	DB *pgx.Conn
}

func NewUserRepository(db *pgx.Conn) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) CreateUser(user *model.User) error {

	_, err := r.DB.Exec(
		context.Background(),
		"INSERT INTO users (username, password) VALUES ($1, $2)",
		user.Username,
		user.Password,
	)

	return err
}

func (r *UserRepository) FindUserByUsername(username string) (*model.User, error) {
	var user model.User

	err := r.DB.QueryRow(
		context.Background(),
		"SELECT id, username, password FROM users WHERE username =$1",
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
