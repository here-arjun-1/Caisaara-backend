package leaderboard

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(30, time.Minute)
	protected.GET("/leaderboard", limiter.Limit, handler.GetLeaderboard)
}
