package bot

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/websocket"
)

type CreateGameRequest struct {
	Level  string `json:"level" binding:"required"`
	Rating int    `json:"rating"`
	Color  string `json:"color"`
}

type PasswordVersionGetter interface {
	GetPasswordVersion(ctx context.Context, userID int64) (int, error)
}

type Handler struct {
	Service  GameService
	Hub      *websocket.Hub
	UserRepo PasswordVersionGetter
}

func NewHandler(service GameService, hub ...*websocket.Hub) *Handler {
	var h *websocket.Hub
	if len(hub) > 0 {
		h = hub[0]
	}
	return &Handler{
		Service: service,
		Hub:     h,
	}
}

func (h *Handler) SetHub(hub *websocket.Hub) {
	h.Hub = hub
}

func (h *Handler) SetUserRepo(repo PasswordVersionGetter) {
	h.UserRepo = repo
}

func (h *Handler) CreateGame(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	var req CreateGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	ctx := c.Request.Context()
	game, err := h.Service.CreateGame(ctx, userID, req.Level, req.Rating, req.Color)

	if errors.Is(err, ErrInvalidLevel) || errors.Is(err, ErrInvalidRating) || errors.Is(err, ErrInvalidColor) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if errors.Is(err, ErrTooManyActiveGames) {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		slog.ErrorContext(ctx, "create bot game handler failed", "player_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusCreated, game)
}

func (h *Handler) GetGame(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	ctx := c.Request.Context()
	gameID := c.Param("gameID")

	game, err := h.Service.GetGame(ctx, gameID, userID)

	if errors.Is(err, ErrGameNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		slog.ErrorContext(ctx, "get bot game handler failed", "game_id", gameID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, game)
}
