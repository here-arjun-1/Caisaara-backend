package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
)

type VerifyEmailHandler struct {
	VerifyEmailService *service.VerifyEmailService
}

func NewVerifyEmailHandler(
	verifyEmailService *service.VerifyEmailService,
) *VerifyEmailHandler {
	return &VerifyEmailHandler{
		VerifyEmailService: verifyEmailService,
	}
}

func (h *VerifyEmailHandler) VerifyEmail(c *gin.Context) {

	token := c.Query("token")

	err := h.VerifyEmailService.VerifyEmail(token)

	if errors.Is(err, service.ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
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
		"message": "email verified successfully",
	})
}
