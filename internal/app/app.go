package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/account"
	"github.com/here-arjun-1/Caisaara-backend/internal/analysis"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth"
	"github.com/here-arjun-1/Caisaara-backend/internal/bot"
	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
	"github.com/here-arjun-1/Caisaara-backend/internal/club"
	"github.com/here-arjun-1/Caisaara-backend/internal/community"
	"github.com/here-arjun-1/Caisaara-backend/internal/config"
	"github.com/here-arjun-1/Caisaara-backend/internal/friends"
	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/invitation"
	"github.com/here-arjun-1/Caisaara-backend/internal/leaderboard"
	"github.com/here-arjun-1/Caisaara-backend/internal/matchmaking"
	"github.com/here-arjun-1/Caisaara-backend/internal/nearby"
	"github.com/here-arjun-1/Caisaara-backend/internal/notification"
	"github.com/here-arjun-1/Caisaara-backend/internal/player"
	"github.com/here-arjun-1/Caisaara-backend/internal/privacy"
	"github.com/here-arjun-1/Caisaara-backend/internal/puzzle"
	"github.com/here-arjun-1/Caisaara-backend/internal/rating"
	"github.com/here-arjun-1/Caisaara-backend/internal/router"
	"github.com/here-arjun-1/Caisaara-backend/internal/tilt"
	"github.com/here-arjun-1/Caisaara-backend/internal/tournament"
	"github.com/here-arjun-1/Caisaara-backend/internal/websocket"
)

type App struct {
	cfg    *config.Config
	infra  *infra
	router *gin.Engine

	authModule             *auth.Module
	gameService            game.GameService
	ratingService          *rating.Service
	notificationService    *notification.Service
	notificationRepository *notification.Repository
	tournamentRepo         tournament.TournamentRepository
	wsHandler              *websocket.Handler
	matcher                *matchmaking.Matcher
	analysisEngine         *analysis.AnalysisEngine
	analysisProcessor      *analysis.AnalysisProcessor
	botEngine              *bot.Engine
}

func New(cfg *config.Config) (*App, error) {
	inf, err := newInfra(cfg)
	if err != nil {
		return nil, err
	}

	authModule := auth.NewModule(inf.db, inf.redis, inf.taskDistributor, cfg.JWTSecret)
	playerModule := player.NewModule(inf.db, inf.avatarStorage)
	communityModule := community.NewModule(inf.db)

	gameRepository := game.NewRepository(inf.db)
	gameService := game.NewService(gameRepository, inf.redis)
	gameHandler := game.NewHandler(gameService)

	analysisRepo := analysis.NewRepository(inf.db)
	analysisEngine := analysis.NewAnalysisEngine(cfg.StockfishPath)
	analysisService := analysis.NewService(analysisRepo, gameRepository, inf.taskDistributor)
	analysisHandler := analysis.NewHandler(analysisService)
	analysisProcessor := analysis.NewAnalysisProcessor(analysisEngine, analysisRepo, gameRepository)

	botRepository := bot.NewRepository(inf.db)
	botEngine := bot.NewEngine(cfg.StockfishPath)
	botService := bot.NewService(botRepository, botEngine)
	botHandler := bot.NewHandler(botService)

	friendsRepository := friends.NewRepository(inf.db)
	friendsService := friends.NewService(friendsRepository)
	friendsHandler := friends.NewHandler(friendsService)

	notificationRepository := notification.NewRepository(inf.db)
	notificationService := notification.NewService(notificationRepository)
	notificationHandler := notification.NewHandler(notificationService)
	friendsService.SetNotifier(notificationService)

	inviteRepo := invitation.NewRedisInviteRepository(inf.redis)
	invitationService := invitation.NewService(
		inviteRepo,
		gameRepository,
		authModule.UserRepository,
		friendsRepository,
	)
	invitationService.SetNotifier(notificationService)
	communityModule.SetNotifier(notificationService)
	invitationHandler := invitation.NewHandler(invitationService)

	tiltRepository := tilt.NewRepository(inf.db)
	tiltService := tilt.NewService(tiltRepository)
	tiltHandler := tilt.NewHandler(tiltService)

	ratingRepository := rating.NewRepository(inf.db)
	ratingService := rating.NewService(ratingRepository)

	queueRepo := matchmaking.NewRedisQueueRepository(inf.redis)
	matchmakingService := matchmaking.NewService(queueRepo, ratingRepository, tiltService)
	matchmakingHandler := matchmaking.NewHandler(matchmakingService)
	matcher := matchmaking.NewMatcher(queueRepo, gameRepository)

	chatRepo := chat.NewRepository(inf.db)
	chatLimiter := chat.NewRedisRateLimiter(inf.redis, 10, time.Minute)
	chatService := chat.NewService(chatRepo, gameRepository, chatLimiter)
	chatService.SetNotifier(notificationService)

	clubRepository := club.NewRepository(inf.db)
	clubService := club.NewService(clubRepository, chatLimiter)
	clubService.SetNotifier(notificationService)

	wsHub := websocket.NewHub()
	clubHandler := club.NewHandler(clubService, wsHub)
	wsHandler := websocket.NewHandler(wsHub, gameService, chatService)
	wsHandler.SetInviteFinder(invitationService)
	wsHandler.SetWatchInviteService(invitationService)
	wsHandler.SetUserRepo(authModule.UserRepository)
	authModule.SetDisconnector(wsHub)
	clubHandler.SetUserRepo(authModule.UserRepository)
	botHandler.SetUserRepo(authModule.UserRepository)
	botHandler.SetHub(wsHub)
	chatHandler := chat.NewHandler(chatService, wsHandler.BroadcastChatMessage)
	chatHandler.SetDirectBroadcaster(wsHandler.BroadcastDirectMessage)
	notificationService.SetBroadcaster(wsHandler.BroadcastNotification)
	notificationService.SetOnlineChecker(wsHandler.IsUserOnline)
	friendsService.SetOnlineChecker(wsHandler.IsUserOnline)

	tournamentRepo := tournament.NewRepository(inf.db)
	tournamentService := tournament.NewService(tournamentRepo, gameRepository, wsHandler)
	tournamentService.SetNotifier(notificationService)
	tournamentHandler := tournament.NewHandler(tournamentService)
	wsHandler.SetTournamentFinder(tournamentService)

	nearbyRepository := nearby.NewRepository(inf.db)
	nearbyService := nearby.NewService(nearbyRepository)
	nearbyHandler := nearby.NewHandler(nearbyService)

	puzzleRepo := puzzle.NewRepository(inf.db)
	puzzleService := puzzle.NewService(puzzleRepo)
	puzzleHandler := puzzle.NewHandler(puzzleService)

	leaderboardRepository := leaderboard.NewRepository(inf.db)
	leaderboardService := leaderboard.NewService(leaderboardRepository)
	leaderboardHandler := leaderboard.NewHandler(leaderboardService)

	accountRepository := account.NewRepository(inf.db)
	accountService := account.NewService(
		accountRepository,
		gameService,
		botService,
		tournamentService,
		matchmakingService,
		wsHub,
	)
	accountHandler := account.NewHandler(accountService)

	privacyRepo := privacy.NewRepository(inf.db)
	privacyService := privacy.NewService(privacyRepo)
	privacyHandler := privacy.NewHandler(privacyService)

	chatService.SetPrivacyChecker(privacyService)
	invitationService.SetPrivacyChecker(privacyService)
	clubService.SetPrivacyChecker(privacyService)
	tournamentService.SetPrivacyChecker(privacyService)

	gameService.SetGameCompletionListener(func(ctx context.Context, gameID string, result string) {
		if err := ratingService.UpdateRatingsAfterGame(ctx, gameID); err != nil {
			slog.ErrorContext(ctx, "update ratings failed", "game_id", gameID, "error", err)
		}
		if err := tournamentService.OnGameCompleted(ctx, gameID, result); err != nil {
			slog.ErrorContext(ctx, "tournament game completion failed", "game_id", gameID, "error", err)
		}
		_, _ = analysisService.TriggerAnalysis(ctx, gameID, 0)
	})

	gameService.SetGameStateUpdatedListener(func(ctx context.Context, g *game.Game) {
		wsHandler.BroadcastGameState(g)
	})

	a := &App{
		cfg:                    cfg,
		infra:                  inf,
		authModule:             authModule,
		gameService:            gameService,
		ratingService:          ratingService,
		notificationService:    notificationService,
		notificationRepository: notificationRepository,
		tournamentRepo:         tournamentRepo,
		wsHandler:              wsHandler,
		matcher:                matcher,
		analysisEngine:         analysisEngine,
		analysisProcessor:      analysisProcessor,
		botEngine:              botEngine,
	}

	r, err := router.New(
		cfg.JWTSecret,
		authModule,
		playerModule,
		communityModule,
		invitationHandler,
		gameHandler,
		matchmakingHandler,
		wsHandler,
		botHandler,
		clubHandler,
		chatHandler,
		tiltHandler,
		analysisHandler,
		nearbyHandler,
		friendsHandler,
		puzzleHandler,
		leaderboardHandler,
		notificationHandler,
		accountHandler,
		tournamentHandler,
		privacyHandler,
	)
	if err != nil {
		a.Close()
		return nil, fmt.Errorf("setup router: %w", err)
	}
	a.router = r

	return a, nil
}

func (a *App) Close() {
	a.botEngine.Close()
	a.analysisEngine.Close()
	a.infra.close()
}
