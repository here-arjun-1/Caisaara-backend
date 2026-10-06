package game

import "github.com/gin-gonic/gin"

func RegisterRoutes(
	handler *Handler,
	_ *gin.Engine,
	api *gin.RouterGroup,
) {
	games := api.Group("/games")

	games.GET("/history", handler.GetGameHistory)
	games.GET("/:gameID", handler.GetGame)
	games.GET("/:gameID/moves", handler.GetMoves)
}

