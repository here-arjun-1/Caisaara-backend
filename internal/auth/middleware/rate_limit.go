package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type client struct {
	count       int
	windowStart time.Time
}

type FixedWindowLimiter struct {
	mu      sync.Mutex
	clients map[string]*client
	limit   int
	window  time.Duration
}

func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	l := &FixedWindowLimiter{
		clients: make(map[string]*client),
		limit:   limit,
		window:  window,
	}
	go l.cleanup()
	return l
}

func (l *FixedWindowLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cl, ok := l.clients[ip]
	if !ok || now.Sub(cl.windowStart) >= l.window {
		cl = &client{windowStart: now}
		l.clients[ip] = cl
	}

	if cl.count >= l.limit {
		return false
	}
	cl.count++
	return true
}

func (l *FixedWindowLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		for ip, cl := range l.clients {
			if time.Since(cl.windowStart) >= l.window {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *FixedWindowLimiter) Limit(c *gin.Context) {
	if !l.allow(c.ClientIP()) {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "too many requests, please try again later",
		})
		return
	}
	c.Next()
}
