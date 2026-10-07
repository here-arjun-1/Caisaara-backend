package router

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/bot"
	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
	"github.com/here-arjun-1/Caisaara-backend/internal/community"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/invitation"
	"github.com/here-arjun-1/Caisaara-backend/internal/matchmaking"
	"github.com/here-arjun-1/Caisaara-backend/internal/player"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
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
	chatHandler ...*chat.Handler,
) (*gin.Engine, error) {
	r := gin.Default()

	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1", "172.16.0.0/12"}); err != nil {
		return nil, fmt.Errorf("set trusted proxies: %w", err)
	}

	r.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "ok", nil)
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

	if len(chatHandler) > 0 && chatHandler[0] != nil {
		chat.RegisterRoutes(chatHandler[0], protected)
	}

	if wsHandler != nil {
		r.GET("/ws/:gameID", wsHandler.Connect)
		protected.GET("/games/:gameID/ws", wsHandler.Connect)
	}

	return r, nil
}
