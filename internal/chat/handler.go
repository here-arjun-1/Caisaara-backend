package chat

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service ChatService
}

func NewHandler(service ChatService) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) GetMessages(c *gin.Context) {
	ctx := c.Request.Context()
	gameID := c.Param("gameID")

	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	userID, ok := val.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id in context",
		})
		return
	}

	messages, err := h.Service.GetMessages(ctx, gameID, userID)
	if err != nil {
		slog.WarnContext(ctx, "get chat messages failed", "game_id", gameID, "user_id", userID, "error", err)
		if err.Error() == "player is not part of this game" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "player is not part of this game",
			})
			return
		}
		if err.Error() == "game not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "game not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve chat messages",
		})
		return
	}

	c.JSON(http.StatusOK, messages)
}
