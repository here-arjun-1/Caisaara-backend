package player

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/response/player/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/response/player/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/response/player/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	profileHandler *handler.ProfileHandler
}

func NewModule(db *pgxpool.Pool) *Module {
	profileRepository := repository.NewProfileRepository(db)
	profileService := service.NewProfileService(profileRepository)

	return &Module{
		profileHandler: handler.NewProfileHandler(profileService),
	}
}

func (m *Module) RegisterRoutes(r *gin.Engine, protected *gin.RouterGroup) {
	publicProfileLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	getProfileLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	profileUpdateLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)

	r.GET("/players/:username", publicProfileLimiter.Limit, m.profileHandler.GetPublicProfile)

	protected.GET("/profile", getProfileLimiter.Limit, m.profileHandler.GetMyProfile)
	protected.PATCH("/profile", profileUpdateLimiter.Limit, middleware.RequireRegisteredUser(), m.profileHandler.UpdateMyProfile)
}
