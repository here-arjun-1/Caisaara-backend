package report

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type ReportPlayerBody struct {
	ReportedID int64  `json:"reported_id" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
	Details    string `json:"details"`
}

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func (h *Handler) ReportPlayer(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req ReportPlayerBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	reportID, err := h.Service.ReportPlayer(c.Request.Context(), userID, req.ReportedID, req.Reason, req.Details)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "player reported", gin.H{"report_id": reportID})
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCannotReportYourself), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsTooLong):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrReportExists):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "report request failed", "path", c.FullPath(), "error", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
