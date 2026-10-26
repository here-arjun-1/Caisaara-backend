package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/notification"
)

func (h *Handler) ConnectNotifications(c *gin.Context) {
	ctx := context.WithoutCancel(c.Request.Context())
	accessToken := c.Query("token")

	if accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "token required",
		})
		return
	}

	claims, err := token.ValidateToken(accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	if claims.IsGuest || claims.UserID <= 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "registered user required for notifications",
		})
		return
	}

	if !h.validatePasswordVersion(ctx, claims) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or revoked token",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.ErrorContext(ctx, "notifications websocket upgrade failed", "user_id", claims.UserID, "error", err)
		return
	}

	roomKey := "notif:" + strconv.FormatInt(claims.UserID, 10)
	client := newClientFromClaims(conn, claims, roomKey)

	room := h.Hub.GetOrCreateRoom(roomKey)
	room.AddClient(client)

	defer func() {
		room.RemoveClient(client.UserID, client)
		client.Close()
		if room.Count() == 0 {
			h.Hub.RemoveRoom(roomKey)
		}
	}()

	go h.writePump(ctx, client)

	_ = client.SendJSON(gin.H{
		"type":    "connected",
		"user_id": claims.UserID,
	})

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}

func (h *Handler) BroadcastNotification(userID int64, n *notification.Notification) {
	if h.Hub == nil || n == nil {
		return
	}
	roomKey := "notif:" + strconv.FormatInt(userID, 10)
	_ = h.Hub.BroadcastJSON(roomKey, gin.H{
		"type":         "notification",
		"notification": n,
	})
}

func (h *Handler) IsUserOnline(userID int64) bool {
	if h.Hub == nil {
		return false
	}
	room := h.Hub.GetRoom("notif:" + strconv.FormatInt(userID, 10))
	return room != nil && room.HasClient(userID)
}
