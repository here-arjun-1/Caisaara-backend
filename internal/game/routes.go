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
	games.POST("/:gameID/resign", handler.Resign)
	games.POST("/:gameID/draw/offer", handler.OfferDraw)
	games.POST("/:gameID/draw/accept", handler.AcceptDraw)
	games.POST("/:gameID/draw/decline", handler.DeclineDraw)
}
