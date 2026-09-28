package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type RegisterHandler struct {
	RegisterService *service.RegisterService
}

func NewRegisterHandler(registerService *service.RegisterService) *RegisterHandler {
	return &RegisterHandler{
		RegisterService: registerService,
	}
}

func (h *RegisterHandler) Register(c *gin.Context) {

	var req dto.RegisterData

	err := c.ShouldBindJSON(&req)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	err = h.RegisterService.Register(req)

	if errors.Is(err, service.ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, service.ErrUsernameTaken) ||
		errors.Is(err, service.ErrEmailTaken) {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "verification email sent",
	})
}

func (h *RegisterHandler) VerifyRegistration(c *gin.Context) {
	var req dto.VerifyRegistrationData

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	accessToken, refreshToken, err := h.RegisterService.VerifyRegistration(req)

	if errors.Is(err, service.ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, service.ErrUsernameTaken) ||
		errors.Is(err, service.ErrEmailTaken) {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "user registered successfully",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}
