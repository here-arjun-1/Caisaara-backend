package admin

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, adminRoutes *gin.RouterGroup) {
	readLimiter := middleware.NewTokenBucketLimiter(120, time.Minute)
	actionLimiter := middleware.NewTokenBucketLimiter(60, time.Hour)

	adminRoutes.GET("/me", readLimiter.Limit, handler.Me)
	adminRoutes.GET("/stats", readLimiter.Limit, handler.GetStats)
	adminRoutes.GET("/health", readLimiter.Limit, handler.GetHealth)

	adminRoutes.GET("/users", readLimiter.Limit, handler.SearchUsers)
	adminRoutes.GET("/users/:id", readLimiter.Limit, handler.GetUser)
	adminRoutes.POST("/users/:id/ban", actionLimiter.Limit, handler.BanUser)
	adminRoutes.POST("/users/:id/unban", actionLimiter.Limit, handler.UnbanUser)
	adminRoutes.POST("/users/:id/logout", actionLimiter.Limit, handler.ForceLogout)

	adminRoutes.GET("/reports", readLimiter.Limit, handler.ListReports)
	adminRoutes.POST("/reports/:id/close", actionLimiter.Limit, handler.CloseReport)

	adminRoutes.GET("/games", readLimiter.Limit, handler.ListLiveGames)
	adminRoutes.GET("/games/:gameID", readLimiter.Limit, handler.GetGame)
	adminRoutes.POST("/games/:gameID/abort", actionLimiter.Limit, handler.AbortGame)
	adminRoutes.GET("/matchmaking", readLimiter.Limit, handler.GetMatchmakingQueues)

	adminRoutes.DELETE("/posts/:id", actionLimiter.Limit, handler.DeletePost)
	adminRoutes.DELETE("/comments/:id", actionLimiter.Limit, handler.DeleteComment)
	adminRoutes.DELETE("/clubs/:id", actionLimiter.Limit, handler.DeleteClub)
	adminRoutes.POST("/tournaments/:id/cancel", actionLimiter.Limit, handler.CancelTournament)

	adminRoutes.GET("/actions", readLimiter.Limit, handler.ListActions)
}
