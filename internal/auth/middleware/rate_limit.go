package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	key := getLimiterKey(c)
	if !l.allow(key) {
		response.Abort(c, http.StatusTooManyRequests, "too many requests, please try again later")
		return
	}
	c.Next()
}

func getLimiterKey(c *gin.Context) string {
	if uid, ok := GetUserID(c); ok && uid > 0 {
		return "user:" + strconv.FormatInt(uid, 10)
	}
	if guestIDValue, exists := c.Get("guest_id"); exists {
		if gid, ok := guestIDValue.(string); ok && gid != "" {
			return "guest:" + gid
		}
	}
	return "ip:" + c.ClientIP()
}

type emailBody struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

func getEmailFromBody(c *gin.Context) string {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var data emailBody
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		return ""
	}

	email := data.Email
	if email == "" {
		email = data.Username
	}

	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	return email
}

func (l *TokenBucketLimiter) LimitByEmail(c *gin.Context) {
	email := getEmailFromBody(c)
	if email == "" {
		c.Next()
		return
	}

	if !l.allow("email:" + email) {
		response.Abort(c, http.StatusTooManyRequests, "too many requests, please try again later")
		return
	}
	c.Next()
}
