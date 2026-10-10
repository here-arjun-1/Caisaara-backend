package router

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/analysis"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/bot"
	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
	"github.com/here-arjun-1/Caisaara-backend/internal/club"
	"github.com/here-arjun-1/Caisaara-backend/internal/community"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/invitation"
	"github.com/here-arjun-1/Caisaara-backend/internal/matchmaking"
	"github.com/here-arjun-1/Caisaara-backend/internal/nearby"
	"github.com/here-arjun-1/Caisaara-backend/internal/player"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
	"github.com/here-arjun-1/Caisaara-backend/internal/tilt"
	"github.com/here-arjun-1/Caisaara-backend/internal/tournament"
	"github.com/here-arjun-1/Caisaara-backend/internal/websocket"
)

func New(
	jwtSecret string,
	authModule *auth.Module,
	playerModule *player.Module,
	communityModule *community.Module,
	invitationHandler *invitation.Handler,
	gameHandler *game.Handler,
	matchmakingHandler *matchmaking.Handler,
	wsHandler *websocket.Handler,
	botHandler *bot.Handler,
	clubHandler *club.Handler,
	chatHandler *chat.Handler,
	tiltHandler *tilt.Handler,
	analysisHandler *analysis.Handler,
	nearbyHandler *nearby.Handler,
	tournamentHandler ...*tournament.Handler,
) (*gin.Engine, error) {
	r := gin.Default()

	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1", "172.16.0.0/12"}); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}

	r.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "ok", nil)
	})

	r.GET("/.well-known/assetlinks.json", func(c *gin.Context) {
		if _, err := os.Stat("./.well-known/assetlinks.json"); err == nil {
			c.File("./.well-known/assetlinks.json")
			return
		}
		c.JSON(http.StatusOK, []gin.H{
			{
				"relation": []string{"delegate_permission/common.handle_all_urls"},
				"target": gin.H{
					"namespace":    "android_app",
					"package_name": "com.abhinav.caisarra",
					"sha256_cert_fingerprints": []string{
						"6D:5C:8C:3A:29:DC:0C:B1:5D:74:42:85:3D:20:E3:11:76:90:9C:FC:03:7F:E6:4D:2C:3A:85:DC:32:98:A8:E8",
					},
				},
			},
		})
	})

	protected := r.Group("/api")
	protected.Use(middleware.JWTMiddleware(jwtSecret, authModule.UserRepository))

	public := r.Group("")
	public.Use(middleware.OptionalJWTMiddleware(jwtSecret))

	authModule.RegisterRoutes(r, protected)
	playerModule.RegisterRoutes(r, protected)
	communityModule.RegisterRoutes(public, protected)

	if invitationHandler != nil {
		invitation.RegisterRoutes(invitationHandler, r, protected)
	}

	if gameHandler != nil {
		game.RegisterRoutes(gameHandler, r, protected)
	}

	if matchmakingHandler != nil {
		matchmaking.RegisterRoutes(matchmakingHandler, protected)
	}

	if botHandler != nil {
		bot.RegisterRoutes(botHandler, r, protected)
	}

	if clubHandler != nil {
		club.RegisterRoutes(clubHandler, r, protected)
	}

	if chatHandler != nil {
		chat.RegisterRoutes(chatHandler, protected)
	}

	if tiltHandler != nil {
		tilt.RegisterRoutes(tiltHandler, protected)
	}

	if analysisHandler != nil {
		analysis.RegisterRoutes(analysisHandler, protected)
	}

	if nearbyHandler != nil {
		nearby.RegisterRoutes(nearbyHandler, protected)
	}

	if len(tournamentHandler) > 0 && tournamentHandler[0] != nil {
		tournament.RegisterRoutes(tournamentHandler[0], protected, public)
	}

	if wsHandler != nil {
		r.GET("/ws/:gameID", wsHandler.Connect)
		r.GET("/ws/tournament/:tournamentID", wsHandler.ConnectTournament)
		protected.GET("/games/:gameID/ws", wsHandler.Connect)
	}

	return r, nil
}
