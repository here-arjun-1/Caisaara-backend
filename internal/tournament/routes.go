package tournament

import "github.com/gin-gonic/gin"

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup) {
	protected.POST("/tournaments", handler.CreateTournament)
}
