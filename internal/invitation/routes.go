package invitation

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, public *gin.Engine, protected *gin.RouterGroup) {
	createLimiter := middleware.NewTokenBucketLimiter(20, time.Minute)
	actionLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	queryLimiter := middleware.NewTokenBucketLimiter(180, time.Minute)

	protected.POST("/invite", createLimiter.Limit, handler.Create)
	protected.POST("/invite/:code/join", actionLimiter.Limit, handler.Join)
	public.GET("/api/invite/:code", queryLimiter.Limit, handler.Preview)
	public.GET("/play/:code", queryLimiter.Limit, handler.Play)

	protected.GET("/challenges", middleware.RequireRegisteredUser(), queryLimiter.Limit, handler.GetMyChallenges)
	protected.GET("/challenges/sent", middleware.RequireRegisteredUser(), queryLimiter.Limit, handler.GetMySentChallenges)
	protected.POST("/challenges/:code/accept", middleware.RequireRegisteredUser(), actionLimiter.Limit, handler.AcceptChallenge)
	protected.POST("/challenges/:code/decline", middleware.RequireRegisteredUser(), actionLimiter.Limit, handler.DeclineChallenge)
	protected.POST("/challenges/:code/cancel", middleware.RequireRegisteredUser(), actionLimiter.Limit, handler.CancelChallenge)

	protected.POST("/games/:gameID/watch-invite", middleware.RequireRegisteredUser(), actionLimiter.Limit, handler.CreateWatchInvite)
}
