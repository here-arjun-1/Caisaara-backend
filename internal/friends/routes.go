package friends

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(30, time.Minute)

	friends := protected.Group("/friends")
	friends.Use(middleware.RequireRegisteredUser())

	friends.GET("", limiter.Limit, handler.GetFriends)
	friends.GET("/online", limiter.Limit, handler.GetOnlineFriends)
	friends.POST("/requests", limiter.Limit, handler.SendRequest)
	friends.GET("/requests", limiter.Limit, handler.GetPendingRequests)
	friends.POST("/requests/:id/accept", limiter.Limit, handler.AcceptRequest)
	friends.POST("/requests/:id/reject", limiter.Limit, handler.RejectRequest)
	friends.DELETE("/requests/:id", limiter.Limit, handler.CancelRequest)
	friends.DELETE("/:userID", limiter.Limit, handler.RemoveFriend)
	friends.GET("/requests/sent", limiter.Limit, handler.GetSentRequests)
	friends.GET("/status/:username", limiter.Limit, handler.GetFriendshipStatus)
	friends.GET("/blocks", limiter.Limit, handler.GetBlockedUsers)
	friends.POST("/blocks/:userID", limiter.Limit, handler.BlockUser)
	friends.DELETE("/blocks/:userID", limiter.Limit, handler.UnblockUser)
}
