package game

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) GetGame(c *gin.Context) {
	gameID := c.Param("gameID")

	currentGame, err := h.Service.GetGame(
		c.Request.Context(),
		gameID,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "game not found",
		})
		return
	}

	c.JSON(http.StatusOK, currentGame)
}

func (h *Handler) GetMoves(c *gin.Context) {
	gameID := c.Param("gameID")

	moves, err := h.Service.GetMoves(
		c.Request.Context(),
		gameID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get game moves",
		})
		return
	}

	c.JSON(http.StatusOK, moves)
}
