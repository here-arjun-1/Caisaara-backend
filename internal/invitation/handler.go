package invitation

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CreateInviteRequest struct {
	TimeControlMinutes int    `json:"time_control_minutes"`
	IncrementSeconds   int    `json:"increment_seconds"`
	Color              string `json:"color" binding:"required"`
}

type Handler struct {
	Service InvitationService
}

func NewHandler(service InvitationService) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) Create(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	userID, ok := userIDValue.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id",
		})
		return
	}

	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ctx := c.Request.Context()

	invite, err := h.Service.CreateInvite(
		ctx,
		userID,
		req.TimeControlMinutes,
		req.IncrementSeconds,
		strings.ToLower(req.Color),
	)

	if errors.Is(err, ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, ErrInvalidTimeControl) || errors.Is(err, ErrInvalidColor) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"invite_code":          invite.Code,
		"game_id":              invite.GameID,
		"link":                 "https://caisaara.duckdns.org/play/" + invite.Code,
		"time_control_minutes": invite.TimeControlMinutes,
		"increment_seconds":    invite.IncrementSeconds,
		"color":                invite.Color,
		"status":               "waiting",
	})
}

func (h *Handler) Play(c *gin.Context) {
	code := c.Param("code")
	ctx := c.Request.Context()

	invite, err := h.Service.GetInvite(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "invite not found or expired",
		})
		return
	}

	user, err := h.Service.FindInviteCreator(ctx, invite.CreatorID)
	if err != nil {
		slog.ErrorContext(ctx, "find invite creator failed", "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "inviter not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invite_code": code,
		"inviter": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
		"game_id":              invite.GameID,
		"time_control_minutes": invite.TimeControlMinutes,
		"increment_seconds":    invite.IncrementSeconds,
		"color":                invite.Color,
		"status":               "waiting",
	})
}

func (h *Handler) Preview(c *gin.Context) {
	code := c.Param("code")
	ctx := c.Request.Context()

	invite, err := h.Service.GetInvite(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "invite not found or expired",
		})
		return
	}

	user, err := h.Service.FindInviteCreator(ctx, invite.CreatorID)
	if err != nil {
		slog.ErrorContext(ctx, "find invite creator failed", "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "inviter not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"inviter": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
		"game_id":              invite.GameID,
		"time_control_minutes": invite.TimeControlMinutes,
		"increment_seconds":    invite.IncrementSeconds,
		"color":                invite.Color,
		"status":               "waiting",
	})
}

func (h *Handler) Join(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	userID, ok := userIDValue.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid user id",
		})
		return
	}

	code := c.Param("code")
	ctx := c.Request.Context()

	gameID, err := h.Service.JoinInvite(ctx, code, userID)

	if errors.Is(err, ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, ErrInviteNotFound) || errors.Is(err, ErrInviteUsed) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Is(err, ErrSelfJoin) || errors.Is(err, ErrAlreadyJoining) {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game_id": gameID,
		"status":  "active",
	})
}
