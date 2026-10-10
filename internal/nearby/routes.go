package nearby

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	limiter := middleware.NewTokenBucketLimiter(20, time.Minute)

	nearby := protected.Group("/nearby")
	nearby.Use(middleware.RequireRegisteredUser())

	nearby.PUT("/location", limiter.Limit, handler.UpdateLocation)
	nearby.PATCH("/visibility", limiter.Limit, handler.SetNearby)
	nearby.GET("/players", limiter.Limit, handler.GetNearbyPlayers)
}
