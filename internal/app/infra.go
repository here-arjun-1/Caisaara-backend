package app

import (
	"context"
	"fmt"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/database"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/validation"
	"github.com/here-arjun-1/Caisaara-backend/internal/config"
	"github.com/here-arjun-1/Caisaara-backend/internal/storage"
	"github.com/here-arjun-1/Caisaara-backend/migrations"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type infra struct {
	db              *pgxpool.Pool
	redis           *redis.Client
	asynqRedisOpt   asynq.RedisClientOpt
	taskDistributor *asynq.Client
	avatarStorage   *storage.S3Storage
}

func newInfra(cfg *config.Config) (*infra, error) {
	if err := validation.Register(); err != nil {
		return nil, fmt.Errorf("register validators: %w", err)
	}

	avatarStorage, err := storage.NewS3Storage(context.Background(), cfg.S3)
	if err != nil {
		return nil, fmt.Errorf("setup s3 storage: %w", err)
	}

	if err := migrations.Run(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	db, err := database.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	redisClient, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	asynqRedisOpt := asynq.RedisClientOpt{Addr: cfg.RedisURL}

	return &infra{
		db:              db,
		redis:           redisClient,
		asynqRedisOpt:   asynqRedisOpt,
		taskDistributor: asynq.NewClient(asynqRedisOpt),
		avatarStorage:   avatarStorage,
	}, nil
}

func (i *infra) close() {
	_ = i.taskDistributor.Close()
	_ = i.redis.Close()
	i.db.Close()
}
