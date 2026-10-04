package auth

import (
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	UserRepository    *repository.UserRepository
	SessionRepository *repository.SessionRepository
	handlers          Handlers
}

func NewModule(
	db *pgxpool.Pool,
	redisClient *redis.Client,
	taskDistributor *asynq.Client,
	jwtSecret string,
) *Module {
	userRepository := repository.NewUserRepository(db)
	sessionRepository := repository.NewSessionRepository(db)
	passwordResetRepository := repository.NewPasswordResetRepository(db)

	registerService := service.NewRegisterService(
		db,
		userRepository,
		sessionRepository,
		redisClient,
		taskDistributor,
		jwtSecret,
	)
	loginService := service.NewLoginService(
		userRepository,
		sessionRepository,
		jwtSecret,
	)
	refreshService := service.NewRefreshService(
		sessionRepository,
		userRepository,
		jwtSecret,
	)
	logoutService := service.NewLogoutService(sessionRepository)
	passwordResetService := service.NewPasswordResetService(
		userRepository,
		passwordResetRepository,
		sessionRepository,
		redisClient,
		taskDistributor,
	)
	ratingService := service.NewRatingService(userRepository)

	return &Module{
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		handlers: Handlers{
			Register:      handler.NewRegisterHandler(registerService),
			Login:         handler.NewLoginHandler(loginService),
			Refresh:       handler.NewRefreshHandler(refreshService),
			Logout:        handler.NewLogoutHandler(logoutService),
			PasswordReset: handler.NewPasswordResetHandler(passwordResetService),
			Rating:        handler.NewRatingHandler(ratingService),
		},
	}
}
