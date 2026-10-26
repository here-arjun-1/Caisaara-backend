package privacy

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

func RegisterRoutes(h *Handler, protected *gin.RouterGroup) {
	protected.GET("/privacy", middleware.RequireRegisteredUser(), h.GetSettings)
	protected.PUT("/privacy", middleware.RequireRegisteredUser(), h.UpdateSettings)
}
