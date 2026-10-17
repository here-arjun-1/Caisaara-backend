package analysis

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(5, time.Minute)

	games := protected.Group("/games")

	games.POST("/:gameID/analysis", limiter.Limit, handler.TriggerAnalysis)
	games.GET("/:gameID/analysis", handler.GetAnalysis)
	games.GET("/:gameID/analysis/moves", handler.GetMoveAnalyses)
	games.GET("/:gameID/analysis/moves/:moveNumber", handler.GetMoveAnalysisByNumber)
	games.POST("/:gameID/analysis/retry", limiter.Limit, handler.RetryAnalysis)
}
