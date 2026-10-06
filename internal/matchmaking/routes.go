package matchmaking

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	matchmaking := protected.Group("/matchmaking")
	matchmaking.Use(middleware.RequireRegisteredUser())

	matchmaking.POST("/join", handler.Join)
	matchmaking.POST("/cancel", handler.Cancel)
	matchmaking.GET("/status", handler.Status)
}
