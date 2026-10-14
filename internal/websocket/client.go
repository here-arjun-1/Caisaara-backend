package websocket

import (
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

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
