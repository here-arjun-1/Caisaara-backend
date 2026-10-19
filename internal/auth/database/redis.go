package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisLogSilencer struct{}

func (redisLogSilencer) Printf(ctx context.Context, format string, v ...interface{}) {}

func ConnectRedis(redisURL string) (*redis.Client, error) {
	redis.SetLogger(redisLogSilencer{})

	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return client, nil
}
