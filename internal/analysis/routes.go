package analysis

import "github.com/gin-gonic/gin"

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	games := protected.Group("/games")

	games.POST("/:gameID/analyze", handler.TriggerAnalysis)
	games.GET("/:gameID/analysis", handler.GetAnalysis)
}
