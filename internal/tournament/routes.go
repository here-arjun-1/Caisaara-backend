package tournament

import "github.com/gin-gonic/gin"

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup, public ...*gin.RouterGroup) {
	protected.POST("/tournaments", handler.CreateTournament)
	protected.POST("/tournaments/:id/start", handler.StartTournament)
	protected.POST("/tournaments/:id/join", handler.JoinTournament)
	protected.POST("/tournaments/:id/leave", handler.LeaveTournament)
	protected.POST("/tournaments/invite/:code/join", handler.JoinTournamentByInviteCode)
	protected.GET("/tournaments/:id/my-games", handler.GetMyGames)

	prefix := "/tournaments"
	targetGroup := protected
	if len(public) > 0 && public[0] != nil {
		targetGroup = public[0]
		prefix = "/api/tournaments"
	}

	targetGroup.GET(prefix, handler.ListPublicTournaments)
	targetGroup.GET(prefix+"/:id", handler.GetTournamentDetails)
	targetGroup.GET(prefix+"/:id/standings", handler.GetStandings)
	targetGroup.GET(prefix+"/:id/rounds", handler.GetRounds)
	targetGroup.GET(prefix+"/:id/rounds/:roundNumber", handler.GetRounds)
	targetGroup.GET(prefix+"/:id/games", handler.GetGames)
	targetGroup.GET(prefix+"/invite/:code", handler.GetTournamentByInviteCode)
}
