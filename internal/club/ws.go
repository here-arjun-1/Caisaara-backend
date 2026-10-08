package club

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*client]bool
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*client]bool),
	}
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	clubID := c.Param("clubID")

	userID, ok := userIDFromToken(c.Query("token"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing token"})
		return
	}

	history, err := h.Service.GetMessages(ctx, clubID, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	conn.SetReadLimit(4096)

	cl := &client{
		conn: conn,
		send: make(chan []byte, 64),
	}
	h.Hub.join(clubID, cl)
	defer h.Hub.leave(clubID, cl)

	go cl.writePump()

	cl.sendJSON(gin.H{"type": "history", "messages": history})

	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		if msg.Type != "message" {
			cl.sendJSON(gin.H{"type": "error", "message": "unsupported message type"})
			continue
		}

		sent, err := h.Service.SendMessage(ctx, clubID, userID, msg.Message)
		if err != nil {
			cl.sendJSON(gin.H{"type": "error", "message": wsErrorMessage(ctx, err)})
			continue
		}

		h.Hub.Broadcast(clubID, gin.H{"type": "message", "message": sent})
	}
}

func (h *Hub) join(clubID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[clubID] == nil {
		h.rooms[clubID] = make(map[*client]bool)
	}
	h.rooms[clubID][c] = true
}

func (h *Hub) leave(clubID string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.rooms[clubID], c)
	if len(h.rooms[clubID]) == 0 {
		delete(h.rooms, clubID)
	}

	close(c.send)
}

func (h *Hub) Broadcast(clubID string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.rooms[clubID] {
		select {
		case c.send <- data:
		default:
		}
	}
}

func (c *client) writePump() {
	for data := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}

func (c *client) sendJSON(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
	}
}

func userIDFromToken(tokenString string) (int64, bool) {
	if tokenString == "" {
		return 0, false
	}

	claims, err := token.ValidateToken(tokenString)
	if err != nil || claims.IsGuest || claims.UserID == 0 {
		return 0, false
	}

	return claims.UserID, true
}

func wsErrorMessage(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, ErrEmptyMessage),
		errors.Is(err, ErrMessageTooLong),
		errors.Is(err, ErrRateLimited),
		errors.Is(err, ErrNotMember),
		errors.Is(err, ErrClubNotFound):
		return err.Error()
	default:
		slog.ErrorContext(ctx, "club websocket message failed", "error", err)
		return "internal server error"
	}
}
