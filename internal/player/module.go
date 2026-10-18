package player

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	profileHandler *handler.ProfileHandler
	avatarHandler  *handler.AvatarHandler
}

func NewModule(db *pgxpool.Pool, avatarStorage *storage.S3Storage) *Module {
	profileRepository := repository.NewProfileRepository(db)
	profileService := service.NewProfileService(profileRepository)
	avatarService := service.NewAvatarService(profileRepository, avatarStorage)

	return &Module{
		profileHandler: handler.NewProfileHandler(profileService),
		avatarHandler:  handler.NewAvatarHandler(avatarService),
	}
}

func (m *Module) RegisterRoutes(r *gin.Engine, protected *gin.RouterGroup) {
	publicProfileLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	getProfileLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	profileUpdateLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	avatarUploadLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)

	r.GET("/players/:username", publicProfileLimiter.Limit, m.profileHandler.GetPublicProfile)
	r.GET("/players/:username/stats/:mode", publicProfileLimiter.Limit, m.profileHandler.GetModeStats)

	protected.GET("/profile", getProfileLimiter.Limit, m.profileHandler.GetMyProfile)
	protected.PATCH("/profile", profileUpdateLimiter.Limit, middleware.RequireRegisteredUser(), m.profileHandler.UpdateMyProfile)
	protected.POST("/profile/avatar", avatarUploadLimiter.Limit, middleware.RequireRegisteredUser(), m.avatarHandler.UploadPhoto)
}
