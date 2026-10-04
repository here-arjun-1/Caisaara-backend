package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type LoginHandler struct {
	LoginService LoginService
}

func NewLoginHandler(loginService LoginService) *LoginHandler {
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
	username, accessToken, refreshToken, needsRating, err := h.LoginService.Login(c.Request.Context(), req)

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
	SetAccessTokenCookie(c, accessToken)
	SetRefreshTokenCookie(c, refreshToken)

	c.JSON(http.StatusOK, gin.H{
		"message":      "login successful",
		"username":     username,
		"needs_rating": needsRating,
	})
}
