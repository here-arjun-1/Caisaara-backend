package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/here-arjun-1/Caisaara-backend/internal/config"
)

const (
	WriteWait      = 10 * time.Second
	PongWait       = 60 * time.Second
	PingPeriod     = (PongWait * 9) / 10
	MaxMessageSize = 4096
)

var allowedOrigins = map[string]bool{
	config.AppBaseURL: true,
}

var Upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return allowedOrigins[origin]
}

type Client struct {
	Conn            *websocket.Conn
	UserID          int64
	GameID          string
	PasswordVersion int
	IsGuest         bool

	Send chan []byte

	closed atomic.Bool
	once   sync.Once
	done   chan struct{}
}

func NewClient(conn *websocket.Conn, userID int64, gameID string, passwordVersion int, isGuest bool) *Client {
	return &Client{
		Conn:            conn,
		UserID:          userID,
		GameID:          gameID,
		PasswordVersion: passwordVersion,
		IsGuest:         isGuest,
		Send:            make(chan []byte, 256),
		done:            make(chan struct{}),
	}
}

func (c *Client) SafeSend(data []byte) (sent bool) {
	if c.closed.Load() {
		return false
	}
	defer func() {
		if r := recover(); r != nil {
			sent = false
		}
	}()
	select {
	case c.Send <- data:
		return true
	default:
		return false
	}
}

func (c *Client) SendJSON(payload any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	return c.SafeSend(data)
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-c.done:
			return
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) Close() {
	c.once.Do(func() {
		c.closed.Store(true)
		if c.done != nil {
			close(c.done)
		}
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
	})
}

func (c *Client) CloseWithReason(code int, reason string) {
	if c.closed.Load() {
		return
	}
	if c.Conn != nil {
		message := websocket.FormatCloseMessage(code, reason)
		_ = c.Conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
	}
	c.Close()
}

type RateLimiter struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
}

func NewRateLimiter(limit float64, window time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     limit,
		capacity:   limit,
		refillRate: limit / window.Seconds(),
		lastRefill: time.Now(),
	}
}

func (l *RateLimiter) Allow() bool {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.tokens += elapsed * l.refillRate
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	l.lastRefill = now

	if l.tokens < 1.0 {
		return false
	}
	l.tokens -= 1.0
	return true
}

func (l *RateLimiter) allow() bool {
	return l.Allow()
}
