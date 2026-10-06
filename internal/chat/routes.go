package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(h *Handler, protected *gin.RouterGroup) {
	protected.GET("/games/:gameID/messages", h.GetMessages)
	protected.POST("/games/:gameID/messages", h.SendMessage)
}
