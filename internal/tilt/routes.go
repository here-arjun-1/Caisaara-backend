package tilt

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	tilt := protected.Group("/tilt")
	tilt.Use(middleware.RequireRegisteredUser())

	tilt.GET("/status", handler.GetStatus)
}
