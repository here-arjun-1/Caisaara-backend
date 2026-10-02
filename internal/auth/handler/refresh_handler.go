package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type RefreshHandler struct {
	RefreshService *service.RefreshService
}

func NewRefreshHandler(
	refreshService *service.RefreshService,
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "refresh token is required",
		})
		return
	}

	req := dto.RefreshData{
		RefreshToken: refreshTokenFromCookie,
	}

	accessToken, refreshToken, err := h.RefreshService.Refresh(req)

	if errors.Is(err, service.ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	SetRefreshTokenCookie(c, refreshToken)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}
