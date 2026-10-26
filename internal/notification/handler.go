package notification

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) GetNotifications(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	beforeID, _ := strconv.ParseInt(c.Query("before_id"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))

	notifications, err := h.Service.GetNotifications(c.Request.Context(), userID, beforeID, limit)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "notifications fetched", gin.H{"notifications": notifications})
}

func (h *Handler) GetUnreadCount(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	count, err := h.Service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "unread count fetched", gin.H{"count": count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || notificationID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid notification id")
		return
	}

	if err := h.Service.MarkRead(c.Request.Context(), userID, notificationID); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "notification marked as read", nil)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.Service.MarkAllRead(c.Request.Context(), userID); err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "all notifications marked as read", nil)
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotificationNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "notification request failed", "path", c.FullPath(), "error", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
