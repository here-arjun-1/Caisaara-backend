package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type LogoutHandler struct {
	LogoutService LogoutService
}

func NewLogoutHandler(
	logoutService LogoutService,
) *LogoutHandler {

	return &LogoutHandler{
		LogoutService: logoutService,
	}
}

func (h *LogoutHandler) Logout(
	c *gin.Context,
) {

	refreshToken, err := GetRefreshTokenFromCookie(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "refresh token is required")
		return
	}

	req := dto.LogoutData{
		RefreshToken: refreshToken,
	}

	err = h.LogoutService.Logout(c.Request.Context(), req)

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	ClearAccessTokenCookie(c)
	ClearRefreshTokenCookie(c)
	ClearGuestTokenCookie(c)

	response.Success(c, http.StatusOK, "logout successful", nil)
}

func (h *LogoutHandler) LogoutAll(
	c *gin.Context,
) {

	refreshToken, err := GetRefreshTokenFromCookie(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "refresh token is required")
		return
	}

	req := dto.LogoutData{
		RefreshToken: refreshToken,
	}

	err = h.LogoutService.LogoutAll(c.Request.Context(), req)

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	ClearAccessTokenCookie(c)
	ClearRefreshTokenCookie(c)
	ClearGuestTokenCookie(c)

	response.Success(c, http.StatusOK, "all sessions logged out successfully", nil)
}
