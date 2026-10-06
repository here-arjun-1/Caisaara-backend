package chat

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service     ChatService
	Broadcaster func(gameID string, msg *ChatMessage)
}

func NewHandler(service ChatService, broadcaster ...func(gameID string, msg *ChatMessage)) *Handler {
	h := &Handler{
		Service: service,
	}
	if len(broadcaster) > 0 {
		h.Broadcaster = broadcaster[0]
	}
	return h
}

func (h *Handler) SendMessage(c *gin.Context) {
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

	var req SendChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	msg, err := h.Service.SendMessage(ctx, gameID, userID, req.Message)
	if err != nil {
		slog.WarnContext(ctx, "send chat message failed", "game_id", gameID, "user_id", userID, "error", err)
		switch err.Error() {
		case "message cannot be empty", "message exceeds maximum length of 500 characters", "cannot send chat messages in a finished game":
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case "rate limit exceeded, please wait":
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": err.Error(),
			})
		case "game not found":
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case "player is not part of this game":
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to send chat message",
			})
		}
		return
	}

	if h.Broadcaster != nil {
		h.Broadcaster(gameID, msg)
	}

	c.JSON(http.StatusCreated, msg)
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
