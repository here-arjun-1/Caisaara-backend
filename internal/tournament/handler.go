package tournament

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func getUserID(c *gin.Context) (int64, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := val.(int64)
	return id, ok
}

func (h *Handler) CreateTournament(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	var req CreateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ctx := c.Request.Context()
	t, err := h.Service.CreateTournament(ctx, userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidName) ||
			errors.Is(err, ErrInvalidTimeControl) ||
			errors.Is(err, ErrInvalidFormat) ||
			errors.Is(err, ErrInvalidVisibility) ||
			errors.Is(err, ErrInvalidMaxPlayers) ||
			errors.Is(err, ErrInvalidTotalRounds) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		slog.ErrorContext(ctx, "create tournament handler failed", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, ToTournamentResponse(t))
}
