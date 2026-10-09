package analysis

import (
	"log/slog"
	"net/http"
	"strconv"

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

func getUserIDFromContext(c *gin.Context) int64 {
	if val, exists := c.Get("user_id"); exists {
		if id, ok := val.(int64); ok {
			return id
		}
	}
	return 0
}

func (h *Handler) TriggerAnalysis(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()
	userID := getUserIDFromContext(c)

	analysis, err := h.Service.TriggerAnalysis(ctx, gameID, userID)
	if err != nil {
		slog.WarnContext(ctx, "trigger analysis handler failed", "game_id", gameID, "error", err)
		switch err.Error() {
		case "game not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "game is not finished":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case "forbidden: not part of this game":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger game analysis"})
		}
		return
	}

	c.JSON(http.StatusAccepted, analysis)
}

func (h *Handler) GetAnalysis(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()
	userID := getUserIDFromContext(c)

	result, err := h.Service.GetAnalysis(ctx, gameID, userID)
	if err != nil {
		slog.WarnContext(ctx, "get analysis handler failed", "game_id", gameID, "error", err)
		switch err.Error() {
		case "game not found", "analysis not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "forbidden: not part of this game":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get game analysis"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetMoveAnalyses(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()
	userID := getUserIDFromContext(c)

	moves, err := h.Service.GetMoveAnalyses(ctx, gameID, userID)
	if err != nil {
		slog.WarnContext(ctx, "get move analyses handler failed", "game_id", gameID, "error", err)
		switch err.Error() {
		case "game not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "forbidden: not part of this game":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get move analyses"})
		}
		return
	}

	c.JSON(http.StatusOK, moves)
}

func (h *Handler) GetMoveAnalysisByNumber(c *gin.Context) {
	gameID := c.Param("gameID")
	moveNumberStr := c.Param("moveNumber")
	ctx := c.Request.Context()
	userID := getUserIDFromContext(c)

	moveNumber, err := strconv.Atoi(moveNumberStr)
	if err != nil || moveNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid move number"})
		return
	}

	moveAnalysis, err := h.Service.GetMoveAnalysisByNumber(ctx, gameID, moveNumber, userID)
	if err != nil {
		slog.WarnContext(ctx, "get move analysis by number handler failed", "game_id", gameID, "move_number", moveNumber, "error", err)
		switch err.Error() {
		case "game not found", "move analysis not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "forbidden: not part of this game":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get move analysis"})
		}
		return
	}

	c.JSON(http.StatusOK, moveAnalysis)
}

func (h *Handler) RetryAnalysis(c *gin.Context) {
	gameID := c.Param("gameID")
	ctx := c.Request.Context()
	userID := getUserIDFromContext(c)

	analysis, err := h.Service.RetryAnalysis(ctx, gameID, userID)
	if err != nil {
		slog.WarnContext(ctx, "retry analysis handler failed", "game_id", gameID, "error", err)
		switch err.Error() {
		case "game not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "game is not finished":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case "forbidden: not part of this game":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry game analysis"})
		}
		return
	}

	c.JSON(http.StatusAccepted, analysis)
}
