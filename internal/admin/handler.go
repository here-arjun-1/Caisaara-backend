package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type BanUserBody struct {
	Reason string `json:"reason"`
	Days   *int   `json:"days" binding:"required"`
}

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) SearchUsers(c *gin.Context) {
	ctx := c.Request.Context()
	search := c.Query("q")

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	users, err := h.Service.SearchUsers(ctx, search, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "search users failed", "limit", limit, "offset", offset, "error", err)
		response.Error(c, http.StatusInternalServerError, "failed to search users")
		return
	}

	response.Success(c, http.StatusOK, "users fetched", users)
}

func (h *Handler) BanUser(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req BanUserBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.Service.BanUser(c.Request.Context(), adminID, userID, *req.Days, req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user banned", nil)
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCannotBanYourself), errors.Is(err, ErrReasonRequired), errors.Is(err, ErrReasonTooLong), errors.Is(err, ErrInvalidBanDays):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrCannotBanAdmin):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "admin request failed", "path", c.FullPath(), "error", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
