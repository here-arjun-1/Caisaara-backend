package friends

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type SendRequestBody struct {
	UserID int64 `json:"user_id" binding:"required"`
}

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func getUserID(c *gin.Context) (int64, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDValue.(int64)
	return userID, ok
}

func getRequestID(c *gin.Context) (int64, bool) {
	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || requestID <= 0 {
		return 0, false
	}
	return requestID, true
}

func getFriendID(c *gin.Context) (int64, bool) {
	friendID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || friendID <= 0 {
		return 0, false
	}
	return friendID, true
}

func (h *Handler) SendRequest(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req SendRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	requestID, err := h.Service.SendRequest(c.Request.Context(), userID, req.UserID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "friend request sent", gin.H{"request_id": requestID})
}

func (h *Handler) GetPendingRequests(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requests, err := h.Service.GetPendingRequests(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "friend requests fetched", gin.H{"requests": requests})
}

func (h *Handler) AcceptRequest(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requestID, ok := getRequestID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid request id")
		return
	}

	err := h.Service.AcceptRequest(c.Request.Context(), userID, requestID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "friend request accepted", nil)
}

func (h *Handler) RejectRequest(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requestID, ok := getRequestID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid request id")
		return
	}

	err := h.Service.RejectRequest(c.Request.Context(), userID, requestID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "friend request rejected", nil)
}

func (h *Handler) GetFriends(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friends, err := h.Service.GetFriends(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "friends fetched", gin.H{"friends": friends})
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCannotAddYourself):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrRequestNotFound), errors.Is(err, ErrNotFriends):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrRequestExists):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "friends request failed", "path", c.FullPath(), "error", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) CancelRequest(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requestID, ok := getRequestID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid request id")
		return
	}

	err := h.Service.CancelRequest(c.Request.Context(), userID, requestID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "friend request cancelled", nil)
}

func (h *Handler) RemoveFriend(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendID, ok := getFriendID(c)
	if !ok {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	err := h.Service.RemoveFriend(c.Request.Context(), userID, friendID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "friend removed", nil)
}
