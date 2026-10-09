package analysis

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service AnalysisService
}

func NewHandler(service AnalysisService) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) TriggerAnalysis(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	result, err := h.Service.AnalyzeGame(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "trigger analysis handler failed", "game_id", gameID, "error", err)
		switch err.Error() {
		case "game not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "game is not finished":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to analyze game"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetAnalysis(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()

	result, err := h.Service.GetAnalysis(ctx, gameID)
	if err != nil {
		slog.WarnContext(ctx, "get analysis handler failed", "game_id", gameID, "error", err)
		if err.Error() == "analysis not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get game analysis"})
		return
	}

	c.JSON(http.StatusOK, result)
}
