package game

import (
	"log/slog"
	"net/http"
	"strconv"

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

	_, err := h.Service.GetGame(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "get moves game check failed", "game_id", gameID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "game not found",
		})
		return
	}

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

	if moves == nil {
		moves = []GameMove{}
	}

	c.JSON(http.StatusOK, moves)
}

func (h *Handler) GetGameHistory(c *gin.Context) {
	var playerID int64
	if val, exists := c.Get("user_id"); exists {
		if id, ok := val.(int64); ok {
			playerID = id
		}
	}

	if playerID == 0 {
		if param := c.Query("player_id"); param != "" {
			if parsed, err := strconv.ParseInt(param, 10, 64); err == nil {
				playerID = parsed
			}
		}
	}

	if playerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "player id required",
		})
		return
	}

	ctx := c.Request.Context()
	games, err := h.Service.GetPlayerGames(ctx, playerID)
	if err != nil {
		slog.ErrorContext(ctx, "get game history failed", "player_id", playerID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get game history",
		})
		return
	}

	if games == nil {
		games = []Game{}
	}

	c.JSON(http.StatusOK, games)
}

