package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
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
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	username, accessToken, refreshToken, needsRating, err := h.LoginService.Login(c.Request.Context(), req)

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

	response.Success(c, http.StatusOK, "login successful", dto.LoginResponse{
		Username:    username,
		NeedsRating: needsRating,
	})
}
