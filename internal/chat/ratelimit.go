package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, gameID string, userID int64) (bool, error)
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type TokenBucketRateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	capacity   float64
	refillRate float64
	stop       chan struct{}
}

func NewTokenBucketRateLimiter(limit int64, window time.Duration) *TokenBucketRateLimiter {
	limiter := &TokenBucketRateLimiter{
		buckets:    make(map[string]*bucket),
		capacity:   float64(limit),
		refillRate: float64(limit) / window.Seconds(),
		stop:       make(chan struct{}),
	}
	go limiter.cleanup()
	return limiter
}

func NewRedisRateLimiter(client *redis.Client, limit int64, window time.Duration) *TokenBucketRateLimiter {
	return NewTokenBucketRateLimiter(limit, window)
}

func (l *TokenBucketRateLimiter) Stop() {
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
}

func (l *TokenBucketRateLimiter) Allow(ctx context.Context, gameID string, userID int64) (bool, error) {
	key := fmt.Sprintf("%s:%d", gameID, userID)

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= 50000 {
			for k, v := range l.buckets {
				if now.Sub(v.lastRefill) > 5*time.Minute {
					delete(l.buckets, k)
				}
			}
			if len(l.buckets) >= 50000 {
				return false, nil
			}
		}
		b = &bucket{tokens: l.capacity, lastRefill: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * l.refillRate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false, nil
	}
	b.tokens--
	return true, nil
}

func (l *TokenBucketRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			l.mu.Lock()
			now := time.Now()
			for k, b := range l.buckets {
				if now.Sub(b.lastRefill) > 15*time.Minute {
					delete(l.buckets, k)
				}
			}
			l.mu.Unlock()
		}
	}
}
