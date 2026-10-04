package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type RefreshHandler struct {
	RefreshService RefreshService
}

func NewRefreshHandler(
	refreshService RefreshService,
) *RefreshHandler {

	return &RefreshHandler{
		RefreshService: refreshService,
	}
}

func (h *RefreshHandler) Refresh(
	c *gin.Context,
) {

	refreshTokenFromCookie, err := GetRefreshTokenFromCookie(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "refresh token is required")
		return
	}

	req := dto.RefreshData{
		RefreshToken: refreshTokenFromCookie,
	}

	accessToken, refreshToken, err := h.RefreshService.Refresh(c.Request.Context(), req)

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	SetAccessTokenCookie(c, accessToken)
	SetRefreshTokenCookie(c, refreshToken)

	response.Success(c, http.StatusOK, "token refreshed successfully", nil)
}
