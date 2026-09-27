package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type LogoutHandler struct {
	LogoutService *service.LogoutService
}

func NewLogoutHandler(
	logoutService *service.LogoutService,
) *LogoutHandler {

	return &LogoutHandler{
		LogoutService: logoutService,
	}
}

func (h *LogoutHandler) Logout(
	c *gin.Context,
) {

	var req dto.LogoutData

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	err := h.LogoutService.Logout(req)

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

	c.JSON(http.StatusOK, gin.H{
		"message": "logout successful",
	})
}

func (h *LogoutHandler) LogoutAll(
	c *gin.Context,
) {

	var req dto.LogoutData

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	err := h.LogoutService.LogoutAll(req)

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

	c.JSON(http.StatusOK, gin.H{
		"message": "all sessions logged out successfully",
	})
}
