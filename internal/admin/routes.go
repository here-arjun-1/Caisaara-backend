package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup, userRepo *repository.UserRepository) {
	adminRoutes := protected.Group("/admin")
	adminRoutes.Use(middleware.RequireAdmin(userRepo))

	adminRoutes.GET("/stats", handler.GetStats)

	adminRoutes.GET("/users", handler.SearchUsers)
	adminRoutes.GET("/users/:id", handler.GetUser)
	adminRoutes.POST("/users/:id/ban", handler.BanUser)
	adminRoutes.POST("/users/:id/unban", handler.UnbanUser)
	adminRoutes.POST("/users/:id/logout", handler.ForceLogout)

	adminRoutes.GET("/reports", handler.ListReports)
	adminRoutes.POST("/reports/:id/close", handler.CloseReport)

	adminRoutes.DELETE("/posts/:id", handler.DeletePost)
	adminRoutes.DELETE("/comments/:id", handler.DeleteComment)
	adminRoutes.DELETE("/clubs/:id", handler.DeleteClub)
	adminRoutes.POST("/tournaments/:id/cancel", handler.CancelTournament)

	adminRoutes.GET("/actions", handler.ListActions)
}
