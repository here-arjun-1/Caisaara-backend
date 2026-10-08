package club

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, r *gin.Engine, protected *gin.RouterGroup) {
	readLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	createLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)
	membershipLimiter := middleware.NewTokenBucketLimiter(20, time.Minute)

	clubs := protected.Group("/clubs")
	clubs.Use(middleware.RequireRegisteredUser())

	clubs.GET("", readLimiter.Limit, handler.ListClubs)
	clubs.POST("", createLimiter.Limit, handler.CreateClub)
	clubs.GET("/:clubID", readLimiter.Limit, handler.GetClub)

	clubs.POST("/:clubID/join", membershipLimiter.Limit, handler.JoinClub)
	clubs.POST("/:clubID/leave", membershipLimiter.Limit, handler.LeaveClub)

	clubs.GET("/:clubID/members", readLimiter.Limit, handler.GetMembers)
	clubs.GET("/:clubID/leaderboard", readLimiter.Limit, handler.GetLeaderboard)

	clubs.GET("/:clubID/messages", readLimiter.Limit, handler.GetMessages)
	clubs.POST("/:clubID/messages", handler.SendMessage)

	r.GET("/ws/clubs/:clubID", readLimiter.Limit, handler.Connect)
}
