package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type LogoutHandler struct {
	LogoutService *service.LogoutService
}

func NewLogoutHandler(logoutService *service.LogoutService) *LogoutHandler {
	return &LogoutHandler{
		LogoutService: logoutService,
	}
}

func (h *LogoutHandler) Logout(c *gin.Context) {
	var req dto.LogoutData

	err := c.ShouldBindJSON(&req)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erroe": "invalid request body",
		})
		return
	}

	err = h.LogoutService.Logout(req)

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
