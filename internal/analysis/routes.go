package analysis

import "github.com/gin-gonic/gin"

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	games := protected.Group("/games")

	games.POST("/:gameID/analysis", handler.TriggerAnalysis)
	games.GET("/:gameID/analysis", handler.GetAnalysis)
	games.GET("/:gameID/analysis/moves", handler.GetMoveAnalyses)
	games.GET("/:gameID/analysis/moves/:moveNumber", handler.GetMoveAnalysisByNumber)
	games.POST("/:gameID/analysis/retry", handler.RetryAnalysis)
}
