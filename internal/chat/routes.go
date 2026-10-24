package chat

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(h *Handler, protected *gin.RouterGroup) {
	protected.GET("/games/:gameID/messages", h.GetMessages)
	protected.POST("/games/:gameID/messages", h.SendMessage)

	messages := protected.Group("/messages")
	messages.Use(middleware.RequireRegisteredUser())
	{
		messages.GET("", h.GetMessages)
		messages.GET("/:userID", h.GetMessages)
		messages.POST("/:userID", h.SendMessage)
		messages.POST("", h.SendMessage)
	}
}
