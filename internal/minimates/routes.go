package minimates

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, r *gin.Engine, authMiddleware gin.HandlerFunc) {
	limiter := middleware.NewTokenBucketLimiter(120, time.Minute)

	group := r.Group("/mini-mates")
	group.Use(authMiddleware, limiter.Limit)
	{
		group.GET("/overview", handler.GetOverview)
		group.GET("/levels/:level", handler.GetLevel)
		group.POST("/levels/:level/hint", handler.GetHint)
		group.POST("/levels/:level/move", handler.SubmitMove)
	}
}
