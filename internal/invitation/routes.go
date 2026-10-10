package invitation

import "github.com/gin-gonic/gin"

func RegisterRoutes(handler *Handler, public *gin.Engine, protected *gin.RouterGroup) {
	protected.POST("/invite", handler.Create)
	protected.POST("/invite/:code/join", handler.Join)
	public.GET("/api/invite/:code", handler.Preview)
	public.GET("/play/:code", handler.Play)
}
