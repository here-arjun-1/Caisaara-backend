package account

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	deleteLimiter := middleware.NewTokenBucketLimiter(5, 15*time.Minute)

	protected.DELETE("/account", deleteLimiter.Limit, middleware.RequireRegisteredUser(), handler.DeleteAccount)
}
