package notification

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(60, time.Minute)

	notifications := protected.Group("/notifications")
	notifications.Use(middleware.RequireRegisteredUser())

	notifications.GET("", limiter.Limit, handler.GetNotifications)
	notifications.GET("/unread-count", limiter.Limit, handler.GetUnreadCount)
	notifications.POST("/read-all", limiter.Limit, handler.MarkAllRead)
	notifications.POST("/:id/read", limiter.Limit, handler.MarkRead)
	notifications.POST("/devices", limiter.Limit, handler.RegisterDevice)
	notifications.DELETE("/devices", limiter.Limit, handler.UnregisterDevice)
}
