package invitation

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

type CreateInviteRequest struct {
	TimeControlMinutes int    `json:"time_control_minutes"`
	IncrementSeconds   int    `json:"increment_seconds"`
	Color              string `json:"color" binding:"required"`
	OpponentID         int64  `json:"opponent_id"`
	Rated              *bool  `json:"-"`
}

func (r *CreateInviteRequest) UnmarshalJSON(data []byte) error {
	type Alias CreateInviteRequest
	aux := struct {
		*Alias
		RawRated      interface{} `json:"rated"`
		RawGameType   interface{} `json:"game_type"`
		RawType       interface{} `json:"type"`
		RawMode       interface{} `json:"mode"`
		RawRatingType interface{} `json:"rating_type"`
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	parseRated := func(val interface{}) *bool {
		if val == nil {
			return nil
		}
		switch v := val.(type) {
		case bool:
			return &v
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			if s == "rated" || s == "true" || s == "1" {
				b := true
				return &b
			}
			if s == "unrated" || s == "casual" || s == "false" || s == "0" {
				b := false
				return &b
			}
		case float64:
			b := v != 0
			return &b
		}
		return nil
	}

	if b := parseRated(aux.RawRated); b != nil {
		r.Rated = b
	} else if b := parseRated(aux.RawGameType); b != nil {
		r.Rated = b
	} else if b := parseRated(aux.RawType); b != nil {
		r.Rated = b
	} else if b := parseRated(aux.RawMode); b != nil {
		r.Rated = b
	} else if b := parseRated(aux.RawRatingType); b != nil {
		r.Rated = b
	}

	return nil
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
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
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

	isRated := true
	if req.Rated != nil {
		isRated = *req.Rated
	}

	invite, err := h.Service.CreateInvite(
		ctx,
		userID,
		req.TimeControlMinutes,
		req.IncrementSeconds,
		strings.ToLower(req.Color),
		req.OpponentID,
		isRated,
	)

	if errors.Is(err, ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, ErrInvalidTimeControl) || errors.Is(err, ErrInvalidColor) || errors.Is(err, ErrChallengeYourself) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Is(err, ErrLevelNotPicked) || errors.Is(err, ErrPlayerBlocked) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Is(err, ErrOpponentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
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
		"rated":                invite.Rated,
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

	status := invite.Status
	if status == "" {
		status = "waiting"
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
		"status":               status,
		"rated":                invite.Rated,
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

	status := invite.Status
	if status == "" {
		status = "waiting"
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
		"status":               status,
		"rated":                invite.Rated,
	})
}

func (h *Handler) Join(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
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

	if errors.Is(err, ErrNotYourChallenge) || errors.Is(err, ErrLevelNotPicked) || errors.Is(err, ErrPlayerBlocked) {
		c.JSON(http.StatusForbidden, gin.H{
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

func (h *Handler) GetMyChallenges(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	challenges, err := h.Service.GetMyChallenges(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"challenges": challenges,
	})
}

func (h *Handler) DeclineChallenge(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	code := c.Param("code")

	err := h.Service.DeclineChallenge(c.Request.Context(), userID, code)

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

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "challenge declined",
	})
}

func (h *Handler) CreateWatchInvite(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	gameID := c.Param("gameID")

	invite, err := h.Service.CreateWatchInvite(c.Request.Context(), userID, gameID)

	if errors.Is(err, ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, ErrGameNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Is(err, ErrNotYourGame) {
		c.JSON(http.StatusForbidden, gin.H{
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
		"watch_code": invite.Code,
		"game_id":    invite.GameID,
		"link":       "https://caisaara.duckdns.org/watch/" + invite.Code,
	})
}
