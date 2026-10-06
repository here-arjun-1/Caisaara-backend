package chat

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, gameID string, userID int64) (bool, error)
}

type RedisRateLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

func NewRedisRateLimiter(client *redis.Client, limit int64, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (r *RedisRateLimiter) Allow(ctx context.Context, gameID string, userID int64) (bool, error) {
	if r == nil || r.client == nil {
		return true, nil
	}

	key := fmt.Sprintf("chat_ratelimit:%s:%d", gameID, userID)
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		slog.ErrorContext(ctx, "redis rate limit error", "game_id", gameID, "user_id", userID, "error", err)
		return true, nil
	}

	if count == 1 {
		_ = r.client.Expire(ctx, key, r.window).Err()
	}

	if count > r.limit {
		return false, nil
	}

	return true, nil
}
