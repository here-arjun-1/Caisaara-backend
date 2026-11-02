package streak

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) GetMyStreak(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	tz := c.Query("tz")
	if tz == "" {
		tz = c.GetHeader("Timezone")
	}

	res, err := h.Service.GetStreak(c.Request.Context(), userID, tz)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get streak"})
		return
	}

	c.JSON(http.StatusOK, res)
}
