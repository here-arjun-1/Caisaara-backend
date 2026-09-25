package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
)

type RegisterHandler struct {
}

func NewRegisterHandler() *RegisterHandler {
	return &RegisterHandler{}
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

	c.JSON(http.StatusCreated, gin.H{
		"message":  "registration request received",
		"username": req.Username,
	})
}
