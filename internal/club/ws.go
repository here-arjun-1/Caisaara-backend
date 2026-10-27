package club

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/websocket"
)

type wsMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (h *Handler) Connect(c *gin.Context) {
	ctx := c.Request.Context()
	clubID := c.Param("clubID")

	claims, ok := middleware.ClaimsFromToken(c.Query("token"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing token"})
		return
	}
	userID := claims.UserID

	if h.UserRepo != nil && !claims.IsGuest && claims.UserID > 0 {
		currentVersion, err := h.UserRepo.GetPasswordVersion(ctx, claims.UserID)
		if err != nil || claims.PasswordVersion != currentVersion {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or revoked token"})
			return
		}
	}

	history, err := h.Service.GetMessages(ctx, clubID, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	conn, err := websocket.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	roomKey := "club:" + clubID
	cl := websocket.NewClient(conn, claims.UserID, roomKey, claims.PasswordVersion, claims.IsGuest)
	defer cl.Close()

	conn.SetReadLimit(websocket.MaxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(websocket.PongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(websocket.PongWait))
		return nil
	})

	room := h.Hub.GetOrCreateRoom(roomKey)
	room.AddClient(cl)
	defer func() {
		room.RemoveClient(cl.UserID, cl)
		h.Hub.RemoveRoomIfEmpty(roomKey)
	}()

	go cl.WritePump()

	cl.SendJSON(gin.H{"type": "history", "messages": history})

	limiter := websocket.NewRateLimiter(10, time.Second)

	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(websocket.PongWait))

		if !limiter.Allow() {
			cl.SendJSON(gin.H{"type": "error", "message": "rate limit exceeded"})
			continue
		}

		if h.UserRepo != nil && cl.UserID > 0 {
			currentVersion, err := h.UserRepo.GetPasswordVersion(ctx, cl.UserID)
			if err != nil || cl.PasswordVersion != currentVersion {
				cl.SendJSON(gin.H{"type": "error", "message": "invalid or revoked token"})
				return
			}
		}

		if msg.Type != "message" {
			cl.SendJSON(gin.H{"type": "error", "message": "unsupported message type"})
			continue
		}

		sent, err := h.Service.SendMessage(ctx, clubID, userID, msg.Message)
		if err != nil {
			cl.SendJSON(gin.H{"type": "error", "message": wsErrorMessage(ctx, err)})
			continue
		}

		_ = h.Hub.BroadcastJSON(roomKey, gin.H{"type": "message", "message": sent})
	}
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
