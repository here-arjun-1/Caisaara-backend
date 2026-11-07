package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
	"github.com/here-arjun-1/Caisaara-backend/internal/tournament"
)

type BanUserBody struct {
	Reason string `json:"reason"`
	Days   *int   `json:"days" binding:"required"`
}

type CloseReportBody struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

type AbortGameBody struct {
	Reason string `json:"reason"`
}

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		Service: service,
	}
}

func getIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func (h *Handler) Me(c *gin.Context) {
	response.Success(c, http.StatusOK, "admin", gin.H{"is_admin": true})
}

func (h *Handler) GetHealth(c *gin.Context) {
	response.Success(c, http.StatusOK, "health fetched", h.Service.GetHealth(c.Request.Context()))
}

func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.Service.GetStats(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "stats fetched", stats)
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

func (h *Handler) GetUser(c *gin.Context) {
	userID, ok := getIDParam(c)
	if !ok {
		return
	}

	user, err := h.Service.GetUser(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user fetched", user)
}

func (h *Handler) BanUser(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := getIDParam(c)
	if !ok {
		return
	}

	var req BanUserBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.Service.BanUser(c.Request.Context(), adminID, userID, *req.Days, req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user banned", nil)
}

func (h *Handler) UnbanUser(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := getIDParam(c)
	if !ok {
		return
	}

	err := h.Service.UnbanUser(c.Request.Context(), adminID, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user unbanned", nil)
}

func (h *Handler) ForceLogout(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := getIDParam(c)
	if !ok {
		return
	}

	err := h.Service.ForceLogout(c.Request.Context(), adminID, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user logged out", nil)
}

func (h *Handler) ListReports(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	reports, err := h.Service.ListReports(c.Request.Context(), status, limit, offset)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "reports fetched", reports)
}

func (h *Handler) CloseReport(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	reportID, ok := getIDParam(c)
	if !ok {
		return
	}

	var req CloseReportBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.Service.CloseReport(c.Request.Context(), adminID, reportID, req.Status, req.Note)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "report closed", nil)
}

func (h *Handler) DeletePost(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	postID, ok := getIDParam(c)
	if !ok {
		return
	}

	err := h.Service.DeletePost(c.Request.Context(), adminID, postID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "post deleted", nil)
}

func (h *Handler) DeleteComment(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	commentID, ok := getIDParam(c)
	if !ok {
		return
	}

	err := h.Service.DeleteComment(c.Request.Context(), adminID, commentID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "comment deleted", nil)
}

func (h *Handler) DeleteClub(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	err := h.Service.DeleteClub(c.Request.Context(), adminID, c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "club deleted", nil)
}

func (h *Handler) CancelTournament(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	tournamentID, ok := getIDParam(c)
	if !ok {
		return
	}

	err := h.Service.CancelTournament(c.Request.Context(), adminID, tournamentID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "tournament cancelled", nil)
}

func (h *Handler) ListActions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	actions, err := h.Service.ListActions(c.Request.Context(), limit, offset)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "actions fetched", actions)
}

func (h *Handler) ListLiveGames(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	games, err := h.Service.ListLiveGames(c.Request.Context(), limit, offset)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "live games fetched", games)
}

func (h *Handler) GetGame(c *gin.Context) {
	detail, err := h.Service.GetGame(c.Request.Context(), c.Param("gameID"))
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "game fetched", detail)
}

func (h *Handler) AbortGame(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var body AbortGameBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.Service.AbortGame(c.Request.Context(), adminID, c.Param("gameID"), body.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "game aborted", nil)
}

func (h *Handler) GetMatchmakingQueues(c *gin.Context) {
	queues, err := h.Service.ListMatchmakingQueues(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "matchmaking queues fetched", queues)
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCannotBanYourself),
		errors.Is(err, ErrReasonRequired),
		errors.Is(err, ErrReasonTooLong),
		errors.Is(err, ErrInvalidBanDays),
		errors.Is(err, ErrInvalidReportStatus),
		errors.Is(err, ErrInvalidResolution),
		errors.Is(err, ErrNoteTooLong),
		errors.Is(err, ErrInvalidClubID),
		errors.Is(err, ErrAbortReasonRequired):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrCannotBanAdmin):
		response.Error(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrReportNotFound),
		errors.Is(err, ErrPostNotFound),
		errors.Is(err, ErrCommentNotFound),
		errors.Is(err, ErrClubNotFound),
		errors.Is(err, ErrGameNotFound),
		errors.Is(err, tournament.ErrTournamentNotFound):
		response.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrUserNotBanned),
		errors.Is(err, ErrReportAlreadyClosed),
		errors.Is(err, ErrGameAlreadyFinished),
		errors.Is(err, ErrTournamentGame),
		errors.Is(err, tournament.ErrNotOngoing):
		response.Error(c, http.StatusConflict, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "admin request failed", "path", c.FullPath(), "error", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
