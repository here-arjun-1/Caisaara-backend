package admin

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

func (h *Handler) SearchUsers(c *gin.Context) {
	ctx := c.Request.Context()
	search := c.Query("q")

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	users, err := h.Service.SearchUsers(ctx, search, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "search users failed", "limit", limit, "offset", offset, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to search users")
		return
	}

	response.Success(c, http.StatusOK, "users fetched", users)
}
