package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type LoginHandler struct {
	LoginService *service.LoginService
}

func NewLoginHandler(loginService *service.LoginService) *LoginHandler {
	return &LoginHandler{
		LoginService: loginService,
	}
}

func (h *LoginHandler) Login(c *gin.Context) {
	var req dto.LoginData
	err := c.ShouldBindJSON(&req)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	accessToken, err := h.LoginService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":      "login successful",
		"username":     req.Username,
		"access_token": accessToken,
	})

}
