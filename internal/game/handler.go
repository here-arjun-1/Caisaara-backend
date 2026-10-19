package game

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
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
	playerID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
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

func (h *Handler) Resign(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	playerID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	currentGame, err := h.Service.ResignGame(ctx, gameID, playerID)
	if err != nil {
		slog.WarnContext(ctx, "resign game failed", "game_id", gameID, "player_id", playerID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, currentGame)
}

func (h *Handler) Draw(c *gin.Context) {
	h.OfferDraw(c)
}

func (h *Handler) OfferDraw(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	playerID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	currentGame, accepted, _, err := h.Service.OfferDraw(ctx, gameID, playerID)
	if err != nil {
		slog.WarnContext(ctx, "offer draw failed", "game_id", gameID, "player_id", playerID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game":            currentGame,
		"accepted":        accepted,
		"draw_offered_by": playerID,
	})
}

func (h *Handler) AcceptDraw(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	playerID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	currentGame, err := h.Service.AcceptDraw(ctx, gameID, playerID)
	if err != nil {
		slog.WarnContext(ctx, "accept draw failed", "game_id", gameID, "player_id", playerID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, currentGame)
}

func (h *Handler) DeclineDraw(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	playerID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	currentGame, err := h.Service.DeclineDraw(ctx, gameID, playerID)
	if err != nil {
		slog.WarnContext(ctx, "decline draw failed", "game_id", gameID, "player_id", playerID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, currentGame)
}

func (h *Handler) ClaimTimeout(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	currentGame, err := h.Service.ClaimTimeout(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "claim timeout failed", "game_id", gameID, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, currentGame)
}
