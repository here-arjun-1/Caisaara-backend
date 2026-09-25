package handler

import (
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

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "user registered successfully",
		"username": req.Username,
	})
}
