package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type client struct {
	tokens     float64
	lastRefill time.Time
}

type TokenBucketLimiter struct {
	mu         sync.Mutex
	clients    map[string]*client
	capacity   float64
	refillRate float64
	stop       chan struct{}
}

func NewTokenBucketLimiter(limit int, window time.Duration) *TokenBucketLimiter {
	l := &TokenBucketLimiter{
		clients:    make(map[string]*client),
		capacity:   float64(limit),
		refillRate: float64(limit) / window.Seconds(),
		stop:       make(chan struct{}),
	}
	go l.cleanup()
	return l
}

func (l *TokenBucketLimiter) Stop() {
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
}

func (l *TokenBucketLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cl, ok := l.clients[ip]
	if !ok {
		cl = &client{tokens: l.capacity, lastRefill: now}
		l.clients[ip] = cl
	}

	elapsed := now.Sub(cl.lastRefill).Seconds()
	cl.tokens += elapsed * l.refillRate
	if cl.tokens > l.capacity {
		cl.tokens = l.capacity
	}
	cl.lastRefill = now

	if cl.tokens < 1 {
		return false
	}
	cl.tokens--
	return true
}

func (l *TokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			l.mu.Lock()
			for ip, cl := range l.clients {
				if time.Since(cl.lastRefill) > 15*time.Minute {
					delete(l.clients, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}

func (l *TokenBucketLimiter) Limit(c *gin.Context) {
	if !l.allow(c.ClientIP()) {
		response.Abort(c, http.StatusTooManyRequests, "too many requests, please try again later")
		return
	}
	c.Next()
}
