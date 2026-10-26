package chat

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type Handler struct {
	Service           ChatService
	Broadcaster       func(gameID string, msg *ChatMessage)
	DirectBroadcaster func(receiverID int64, msg *DirectMessage)
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

func (h *Handler) SetDirectBroadcaster(broadcaster func(receiverID int64, msg *DirectMessage)) {
	h.DirectBroadcaster = broadcaster
}

func (h *Handler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SendChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if gameID := c.Param("gameID"); gameID != "" {
		msg, err := h.Service.SendMessage(ctx, gameID, userID, req.Message)
		if err != nil {
			slog.WarnContext(ctx, "send chat message failed", "game_id", gameID, "user_id", userID, "error", err)
			switch err.Error() {
			case "message cannot be empty", "message exceeds maximum length of 500 characters", "cannot send chat messages in a finished game":
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			case "rate limit exceeded, please wait":
				c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			case "game not found":
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			case "player is not part of this game":
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send chat message"})
			}
			return
		}
		if h.Broadcaster != nil {
			h.Broadcaster(gameID, msg)
		}
		c.JSON(http.StatusCreated, msg)
		return
	}

	var receiverID int64
	if target := c.Param("userID"); target != "" {
		receiverID, _ = strconv.ParseInt(target, 10, 64)
	}
	if receiverID <= 0 {
		receiverID = req.ReceiverID
	}
	if receiverID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	msg, err := h.Service.SendDirectMessage(ctx, userID, receiverID, strings.TrimSpace(req.Message))
	if err != nil {
		slog.WarnContext(ctx, "send direct message failed", "sender_id", userID, "receiver_id", receiverID, "error", err)
		switch err.Error() {
		case "cannot send message to yourself", "message cannot be empty", "message exceeds maximum length of 500 characters":
			response.Error(c, http.StatusBadRequest, err.Error())
		case "recipient user not found":
			response.Error(c, http.StatusNotFound, err.Error())
		case "rate limit exceeded, please wait":
			response.Error(c, http.StatusTooManyRequests, err.Error())
		case "not allowed by privacy settings":
			response.Error(c, http.StatusForbidden, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to send message")
		}
		return
	}

	if h.DirectBroadcaster != nil {
		h.DirectBroadcaster(receiverID, msg)
	}

	c.JSON(http.StatusCreated, msg)
}

func (h *Handler) GetMessages(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if gameID := c.Param("gameID"); gameID != "" {
		messages, err := h.Service.GetMessages(ctx, gameID, userID)
		if err != nil {
			slog.WarnContext(ctx, "get chat messages failed", "game_id", gameID, "user_id", userID, "error", err)
			if err.Error() == "player is not part of this game" {
				c.JSON(http.StatusForbidden, gin.H{"error": "player is not part of this game"})
				return
			}
			if err.Error() == "game not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve chat messages"})
			return
		}
		c.JSON(http.StatusOK, messages)
		return
	}

	if target := c.Param("userID"); target != "" {
		otherID, err := strconv.ParseInt(target, 10, 64)
		if err != nil || otherID <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid user id")
			return
		}
		messages, err := h.Service.GetDirectMessages(ctx, userID, otherID)
		if err != nil {
			if err.Error() == "invalid user id" {
				response.Error(c, http.StatusBadRequest, "invalid user id")
				return
			}
			response.Error(c, http.StatusInternalServerError, "failed to get messages")
			return
		}
		response.Success(c, http.StatusOK, "messages fetched", gin.H{"messages": messages})
		return
	}

	conversations, err := h.Service.GetConversations(ctx, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get conversations")
		return
	}
	response.Success(c, http.StatusOK, "conversations fetched", gin.H{"conversations": conversations})
}
