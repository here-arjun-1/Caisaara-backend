package club

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/websocket"
)

type CreateClubRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type SendMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type PasswordVersionGetter interface {
	GetPasswordVersion(ctx context.Context, userID int64) (int, error)
}

type Handler struct {
	Service  ClubService
	Hub      *websocket.Hub
	UserRepo PasswordVersionGetter
}

func NewHandler(service ClubService, hub ...*websocket.Hub) *Handler {
	var h *websocket.Hub
	if len(hub) > 0 && hub[0] != nil {
		h = hub[0]
	} else {
		h = websocket.NewHub()
	}
	return &Handler{
		Service: service,
		Hub:     h,
	}
}

func (h *Handler) SetUserRepo(repo PasswordVersionGetter) {
	h.UserRepo = repo
}

func (h *Handler) CreateClub(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req CreateClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	club, err := h.Service.CreateClub(c.Request.Context(), userID, req.Name, req.Description)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, club)
}

func (h *Handler) ListClubs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	clubs, err := h.Service.ListClubs(c.Request.Context(), limit, offset)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, clubs)
}

func (h *Handler) GetClub(c *gin.Context) {
	club, err := h.Service.GetClub(c.Request.Context(), c.Param("clubID"))
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, club)
}

func (h *Handler) JoinClub(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if err := h.Service.JoinClub(c.Request.Context(), c.Param("clubID"), userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "joined club"})
}

func (h *Handler) LeaveClub(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	if err := h.Service.LeaveClub(c.Request.Context(), c.Param("clubID"), userID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left club"})
}

func (h *Handler) GetMembers(c *gin.Context) {
	members, err := h.Service.GetMembers(c.Request.Context(), c.Param("clubID"))
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, members)
}

func (h *Handler) GetLeaderboard(c *gin.Context) {
	entries, err := h.Service.GetLeaderboard(c.Request.Context(), c.Param("clubID"))
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (h *Handler) GetMessages(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	messages, err := h.Service.GetMessages(c.Request.Context(), c.Param("clubID"), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *Handler) SendMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	clubID := c.Param("clubID")
	msg, err := h.Service.SendMessage(c.Request.Context(), clubID, userID, req.Message)
	if err != nil {
		handleError(c, err)
		return
	}

	_ = h.Hub.BroadcastJSON("club:"+clubID, gin.H{"type": "message", "message": msg})

	c.JSON(http.StatusCreated, msg)
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrClubNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotMember):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ErrClubNameTaken), errors.Is(err, ErrAlreadyMember):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidClubName),
		errors.Is(err, ErrDescriptionTooLong),
		errors.Is(err, ErrOwnerCannotLeave),
		errors.Is(err, ErrEmptyMessage),
		errors.Is(err, ErrMessageTooLong),
		errors.Is(err, ErrTargetNotAdmin):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		slog.ErrorContext(c.Request.Context(), "club request failed", "path", c.FullPath(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func getTargetID(c *gin.Context) (int64, bool) {
	targetID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil {
		return 0, false
	}
	return targetID, true
}

func (h *Handler) PromoteToAdmin(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	targetID, ok := getTargetID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if err := h.Service.PromoteToAdmin(c.Request.Context(), c.Param("clubID"), userID, targetID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member is now an admin"})
}

func (h *Handler) RemoveAdmin(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	targetID, ok := getTargetID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if err := h.Service.RemoveAdmin(c.Request.Context(), c.Param("clubID"), userID, targetID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin removed"})
}

func (h *Handler) KickMember(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	targetID, ok := getTargetID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	clubID := c.Param("clubID")
	if err := h.Service.KickMember(c.Request.Context(), clubID, userID, targetID); err != nil {
		handleError(c, err)
		return
	}

	h.Hub.DisconnectUserFromRoom("club:"+clubID, targetID)

	c.JSON(http.StatusOK, gin.H{"message": "member removed from club"})
}

func (h *Handler) TransferLeadership(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}
	targetID, ok := getTargetID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if err := h.Service.TransferLeadership(c.Request.Context(), c.Param("clubID"), userID, targetID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "leadership transferred"})
}
