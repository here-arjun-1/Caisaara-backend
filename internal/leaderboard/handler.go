package leaderboard

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

func (h *Handler) GetLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()

	mode := c.Query("mode")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	entries, err := h.Service.GetLeaderboard(ctx, mode, limit, offset)
	if errors.Is(err, ErrInvalidMode) {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, "get leaderboard failed", "mode", mode, "limit", limit, "offset", offset, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to get leaderboard")
		return
	}

	response.Success(c, http.StatusOK, "leaderboard fetched", entries)
}

func (h *Handler) GetLocalLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()

	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	mode := c.Query("mode")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	radiusKm := 0
	if radiusText := c.Query("radius_km"); radiusText != "" {
		value, err := strconv.Atoi(radiusText)
		if err != nil {
			response.Error(c, http.StatusBadRequest, ErrInvalidRadius.Error())
			return
		}
		radiusKm = value
	}

	entries, err := h.Service.GetLocalLeaderboard(ctx, userID, mode, radiusKm, limit, offset)
	switch {
	case errors.Is(err, ErrInvalidMode), errors.Is(err, ErrInvalidRadius), errors.Is(err, ErrLocationNotSaved), errors.Is(err, ErrLocationTooOld):
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, ErrNearbyOff):
		response.Error(c, http.StatusForbidden, err.Error())
		return
	case err != nil:
		slog.ErrorContext(ctx, "get local leaderboard failed", "user_id", userID, "mode", mode, "radius_km", radiusKm, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to get local leaderboard")
		return
	}

	response.Success(c, http.StatusOK, "local leaderboard fetched", entries)
}
