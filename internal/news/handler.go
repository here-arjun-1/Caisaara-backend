package news

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

func (h *Handler) GetNews(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))

	articles, err := h.Service.GetNews(c.Request.Context(), page)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "get news failed", "page", page, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to fetch news")
		return
	}

	response.Success(c, http.StatusOK, "news fetched", gin.H{"articles": articles})
}
