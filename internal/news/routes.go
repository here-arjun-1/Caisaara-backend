package news

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(60, time.Minute)

	news := protected.Group("/news")

	news.GET("", limiter.Limit, handler.GetNews)
}
