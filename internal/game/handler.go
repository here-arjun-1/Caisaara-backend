package game

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service GameService
}

func NewHandler(service GameService) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) GetGame(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	currentGame, err := h.Service.GetGame(
		ctx,
		gameID,
	)
	if err != nil {
		slog.WarnContext(ctx, "get game failed", "game_id", gameID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "game not found",
		})
		return
	}

	c.JSON(http.StatusOK, currentGame)
}

func (h *Handler) GetMoves(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	moves, err := h.Service.GetMoves(
		ctx,
		gameID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get moves handler failed", "game_id", gameID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get game moves",
		})
		return
	}

	c.JSON(http.StatusOK, moves)
}
