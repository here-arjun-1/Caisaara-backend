package streak

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(
	handler *Handler,
	r *gin.Engine,
	authMiddleware gin.HandlerFunc,
) {
	streakLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)

	if r != nil && authMiddleware != nil {
		r.GET("/streak", authMiddleware, streakLimiter.Limit, handler.GetMyStreak)
	}
}
