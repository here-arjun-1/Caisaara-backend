package minimates

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type Handler struct {
	Service MiniMatesService
}

func NewHandler(service MiniMatesService) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	res, err := h.Service.GetOverview(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "get mini-mates overview failed", "user_id", userID, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to get mini-mates overview")
		return
	}

	response.Success(c, http.StatusOK, "mini-mates overview fetched", res)
}

func (h *Handler) GetLevel(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	levelStr := c.Param("level")
	levelNumber, err := strconv.Atoi(levelStr)
	if err != nil || levelNumber <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid level number")
		return
	}

	res, err := h.Service.GetLevel(ctx, userID, levelNumber)
	if err != nil {
		if errors.Is(err, ErrLevelLocked) {
			response.Error(c, http.StatusForbidden, "level is locked")
			return
		}
		if errors.Is(err, ErrLevelNotFound) {
			response.Error(c, http.StatusNotFound, "level not found")
			return
		}
		slog.ErrorContext(ctx, "get mini-mates level failed", "user_id", userID, "level", levelNumber, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to get level")
		return
	}

	response.Success(c, http.StatusOK, "mini-mates level fetched", res)
}

func (h *Handler) GetHint(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	levelStr := c.Param("level")
	levelNumber, err := strconv.Atoi(levelStr)
	if err != nil || levelNumber <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid level number")
		return
	}

	var req MiniMatesHintRequest
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&req)
	}
	if moveIndexStr := c.Query("move_index"); moveIndexStr != "" && req.MoveIndex == 0 {
		if idx, err := strconv.Atoi(moveIndexStr); err == nil && idx >= 0 {
			req.MoveIndex = idx
		}
	}

	res, err := h.Service.GetHint(ctx, userID, levelNumber, req.MoveIndex)
	if err != nil {
		if errors.Is(err, ErrLevelLocked) {
			response.Error(c, http.StatusForbidden, "level is locked")
			return
		}
		if errors.Is(err, ErrLevelNotFound) {
			response.Error(c, http.StatusNotFound, "level not found")
			return
		}
		slog.ErrorContext(ctx, "get mini-mates hint failed", "user_id", userID, "level", levelNumber, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to get hint")
		return
	}

	response.Success(c, http.StatusOK, "hint fetched", res)
}

func (h *Handler) SubmitMove(c *gin.Context) {
	ctx := c.Request.Context()
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	levelStr := c.Param("level")
	levelNumber, err := strconv.Atoi(levelStr)
	if err != nil || levelNumber <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid level number")
		return
	}

	var req SubmitMiniMatesMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if moveIndexStr := c.Query("move_index"); moveIndexStr != "" && req.MoveIndex == 0 {
		if idx, err := strconv.Atoi(moveIndexStr); err == nil && idx >= 0 {
			req.MoveIndex = idx
		}
	}
	if moveIndexStr := c.Query("moveIndex"); moveIndexStr != "" && req.MoveIndex == 0 {
		if idx, err := strconv.Atoi(moveIndexStr); err == nil && idx >= 0 {
			req.MoveIndex = idx
		}
	}
	if stepStr := c.Query("step"); stepStr != "" && req.Step == nil {
		if s, err := strconv.Atoi(stepStr); err == nil && s >= 1 {
			req.Step = &s
		}
	}

	res, err := h.Service.SubmitMove(ctx, userID, levelNumber, req)
	if err != nil {
		if errors.Is(err, ErrLevelLocked) {
			response.Error(c, http.StatusForbidden, "level is locked")
			return
		}
		if errors.Is(err, ErrLevelNotFound) {
			response.Error(c, http.StatusNotFound, "level not found")
			return
		}
		slog.ErrorContext(ctx, "submit mini-mates move failed", "user_id", userID, "level", levelNumber, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to submit move")
		return
	}

	response.Success(c, http.StatusOK, "move processed", res)
}
