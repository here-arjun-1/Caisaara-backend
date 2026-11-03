package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup, userRepo *repository.UserRepository) {
	adminRoutes := protected.Group("/admin")
	adminRoutes.Use(middleware.RequireAdmin(userRepo))

	adminRoutes.GET("/users", handler.SearchUsers)
}
