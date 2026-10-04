package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	migrations := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_version INT DEFAULT 1;`,
		`CREATE TABLE IF NOT EXISTS player_profiles (
			user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			display_name VARCHAR(255),
			country VARCHAR(100),
			bio TEXT,
			avatar_url TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range migrations {
		_, _ = pool.Exec(context.Background(), query)
	}

	return pool, nil
}
