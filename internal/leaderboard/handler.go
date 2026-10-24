package leaderboard

import (
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

func (h *Handler) GetDailyLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	entries, err := h.Service.GetDailyLeaderboard(ctx, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "get daily leaderboard failed", "limit", limit, "offset", offset, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to get leaderboard")
		return
	}

	response.Success(c, http.StatusOK, "leaderboard fetched", entries)
}
