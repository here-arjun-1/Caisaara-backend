package invitation

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(handler *Handler, public *gin.Engine, protected *gin.RouterGroup) {
	protected.POST("/invite", handler.Create)
	protected.POST("/invite/:code/join", handler.Join)
	public.GET("/api/invite/:code", handler.Preview)
	public.GET("/play/:code", handler.Play)

	protected.GET("/challenges", middleware.RequireRegisteredUser(), handler.GetMyChallenges)
	protected.POST("/challenges/:code/decline", middleware.RequireRegisteredUser(), handler.DeclineChallenge)
}
