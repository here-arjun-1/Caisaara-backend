package report

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(10, time.Hour)

	reports := protected.Group("/reports")
	reports.Use(middleware.RequireRegisteredUser())

	reports.POST("", limiter.Limit, handler.ReportPlayer)
}
