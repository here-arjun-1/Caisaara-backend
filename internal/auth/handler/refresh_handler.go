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

	var req dto.RefreshData

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	accessToken, err := h.RefreshService.Refresh(req)

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
		"access_token": accessToken,
	})
}
