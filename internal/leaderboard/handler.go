package leaderboard

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
