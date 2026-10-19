package puzzle

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(handler *Handler, r *gin.Engine, authMiddleware gin.HandlerFunc) {
	puzzles := r.Group("/puzzles")
	puzzles.Use(authMiddleware)
	{
		puzzles.GET("/daily", handler.GetDailyPuzzle)
		puzzles.POST("/daily/attempt", handler.SubmitAttempt)
		puzzles.POST("/daily/hint", handler.GetHint)
		puzzles.POST("/daily/solution", handler.GetSolution)
		puzzles.GET("/next", handler.GetNextPuzzle)
		puzzles.GET("/stats", handler.GetUserStats)
		puzzles.GET("/history", handler.GetUserHistory)

		rush := puzzles.Group("/rush")
		{
			rush.POST("/start", handler.StartRush)
			rush.POST("/move", handler.SubmitRushMove)
			rush.POST("/end", handler.EndRush)
			rush.GET("/leaderboard", handler.GetRushLeaderboard)
			rush.GET("/stats", handler.GetUserRushStats)
		}
	}
}
