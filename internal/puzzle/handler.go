package puzzle

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidRequestBody  = errors.New("invalid request body")
	ErrInvalidMode         = errors.New("invalid mode: must be 3min, 5min, or survival")
	ErrSessionNotFound     = errors.New("rush session not found")
	ErrUnauthorizedSession = errors.New("unauthorized session access")
	ErrPuzzleMismatch      = errors.New("puzzle id does not match current session puzzle")
	ErrInvalidMoves        = errors.New("invalid puzzle moves sequence")
	ErrPuzzleNotFound      = errors.New("puzzle not found")
)

type Handler struct {
	Service PuzzleService
}

func NewHandler(service PuzzleService) *Handler {
	return &Handler{
		Service: service,
	}
}

func handlePuzzleError(c *gin.Context, err error, action string) {
	switch {
	case errors.Is(err, ErrPuzzleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrPuzzleNotFound.Error()})
	case errors.Is(err, ErrSessionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": ErrSessionNotFound.Error()})
	case errors.Is(err, ErrUnauthorizedSession):
		c.JSON(http.StatusForbidden, gin.H{"error": ErrUnauthorizedSession.Error()})
	case errors.Is(err, ErrInvalidMode):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidMode.Error()})
	case errors.Is(err, ErrPuzzleMismatch):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrPuzzleMismatch.Error()})
	case errors.Is(err, ErrInvalidMoves):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidMoves.Error()})
	default:
		slog.ErrorContext(c.Request.Context(), action+" failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to " + action})
	}
}

func (h *Handler) GetDailyPuzzle(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	res, err := h.Service.GetDailyPuzzle(ctx, userID)
	if err != nil {
		handlePuzzleError(c, err, "get daily puzzle")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) SubmitAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req SubmitAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.SubmitAttempt(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "submit attempt")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetHint(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req HintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.GetHint(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "get hint")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetSolution(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req SolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.GetSolution(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "get solution")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetNextPuzzle(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	theme := c.Query("theme")

	res, err := h.Service.GetNextPuzzle(ctx, userID, theme)
	if err != nil {
		handlePuzzleError(c, err, "get puzzle")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetUserStats(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	res, err := h.Service.GetUserStats(ctx, userID)
	if err != nil {
		handlePuzzleError(c, err, "get puzzle stats")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetUserHistory(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	res, err := h.Service.GetUserHistory(ctx, userID, limit, offset)
	if err != nil {
		handlePuzzleError(c, err, "get puzzle history")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) StartRush(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req StartRushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.StartRush(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "start rush")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) SubmitRushMove(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req SubmitRushMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.SubmitRushMove(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "submit rush move")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) EndRush(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req EndRushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.EndRush(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "end rush")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetRushLeaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	mode := c.DefaultQuery("mode", "3min")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}

	res, err := h.Service.GetRushLeaderboard(ctx, mode, limit)
	if err != nil {
		handlePuzzleError(c, err, "get leaderboard")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetUserRushStats(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	res, err := h.Service.GetUserRushStats(ctx, userID)
	if err != nil {
		handlePuzzleError(c, err, "get rush stats")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetUserRushHistory(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	mode := c.Query("mode")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	res, err := h.Service.GetUserRushHistory(ctx, userID, mode, limit, offset)
	if err != nil {
		handlePuzzleError(c, err, "get rush history")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetThemes(c *gin.Context) {
	ctx := c.Request.Context()
	res, err := h.Service.GetThemes(ctx)
	if err != nil {
		handlePuzzleError(c, err, "get puzzle themes")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetCustomPuzzle(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var themes []string
	if t := strings.TrimSpace(c.Query("theme")); t != "" {
		themes = append(themes, t)
	}
	if ts := strings.TrimSpace(c.Query("themes")); ts != "" {
		for _, part := range strings.Split(ts, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				themes = append(themes, part)
			}
		}
	}

	minRating, _ := strconv.Atoi(c.DefaultQuery("min_rating", "0"))
	maxRating, _ := strconv.Atoi(c.DefaultQuery("max_rating", "0"))
	excludeSolvedStr := c.DefaultQuery("exclude_solved", "true")
	excludeSolved := excludeSolvedStr == "true" || excludeSolvedStr == "1"
	reviewMistakesStr := c.DefaultQuery("review_mistakes", "false")
	reviewMistakes := reviewMistakesStr == "true" || reviewMistakesStr == "1"

	filter := CustomPuzzleFilter{
		UserID:         userID,
		Themes:         themes,
		MinRating:      minRating,
		MaxRating:      maxRating,
		ExcludeSolved:  excludeSolved,
		ReviewMistakes: reviewMistakes,
	}

	res, err := h.Service.GetCustomPuzzle(ctx, userID, filter)
	if err != nil {
		handlePuzzleError(c, err, "get custom puzzle")
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handler) SubmitCustomAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
		return
	}

	var req SubmitCustomAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequestBody.Error()})
		return
	}

	res, err := h.Service.SubmitCustomAttempt(ctx, userID, req)
	if err != nil {
		handlePuzzleError(c, err, "submit custom attempt")
		return
	}

	c.JSON(http.StatusOK, res)
}
