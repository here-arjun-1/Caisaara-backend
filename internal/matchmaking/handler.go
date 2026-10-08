package matchmaking

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type JoinRequest struct {
	TimeControlMinutes int  `json:"time_control_minutes" binding:"required"`
	Rated              bool `json:"rated"`
	IgnoreTilt         bool `json:"ignore_tilt"`
}

type Handler struct {
	Service MatchmakingService
}

func NewHandler(service MatchmakingService) *Handler {
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

func (h *Handler) Join(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	var req JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	tiltStatus, err := h.Service.Join(c.Request.Context(), userID, req.TimeControlMinutes, req.Rated, req.IgnoreTilt)

	if errors.Is(err, ErrTilted) {
		c.JSON(http.StatusConflict, gin.H{
			"error":        err.Error(),
			"tilt_warning": tiltStatus,
		})
		return
	}

	if errors.Is(err, ErrInvalidTimeControl) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Is(err, ErrAlreadyInQueue) {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"state": StatusSearching,
	})
}

func (h *Handler) Cancel(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	err := h.Service.Cancel(c.Request.Context(), userID)

	if errors.Is(err, ErrNotInQueue) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"state": StatusIdle,
	})
}

func (h *Handler) Status(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	status, err := h.Service.GetStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}
