package minimates

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(120, time.Minute)

	group := protected.Group("/mini-mates")
	group.Use(limiter.Limit)
	{
		group.GET("", handler.GetOverview)
		group.GET("/overview", handler.GetOverview)
		group.GET("/levels/:level", handler.GetLevel)
		group.POST("/levels/:level/move", handler.SubmitMove)
	}
}

