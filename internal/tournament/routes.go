package tournament

import "github.com/gin-gonic/gin"

func RegisterRoutes(handler *Handler, protected *gin.RouterGroup, public ...*gin.RouterGroup) {
	protected.GET("/tournaments", handler.ListPublicTournaments)
	protected.GET("/tournaments/:id", handler.GetTournamentDetails)
	protected.GET("/tournaments/invite/:code", handler.GetTournamentByInviteCode)
	protected.POST("/tournaments/:id/join", handler.JoinTournament)
	protected.POST("/tournaments", handler.CreateTournament)

	if len(public) > 0 && public[0] != nil {
		public[0].GET("/api/tournaments", handler.ListPublicTournaments)
		public[0].GET("/api/tournaments/:id", handler.GetTournamentDetails)
		public[0].GET("/api/tournaments/invite/:code", handler.GetTournamentByInviteCode)
	}
}
