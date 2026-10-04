package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
)

type Handlers struct {
	Register      *handler.RegisterHandler
	Login         *handler.LoginHandler
	Refresh       *handler.RefreshHandler
	Logout        *handler.LogoutHandler
	PasswordReset *handler.PasswordResetHandler
	Rating        *handler.RatingHandler
}

func (m *Module) RegisterRoutes(r *gin.Engine, protected *gin.RouterGroup) {
	h := m.handlers

	loginLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	registerLimiter := middleware.NewTokenBucketLimiter(10, time.Hour)
	verifyRegistrationLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)
	guestLoginLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	forgotLimiter := middleware.NewTokenBucketLimiter(5, 15*time.Minute)
	verifyResetLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)
	resetPasswordLimiter := middleware.NewTokenBucketLimiter(5, 15*time.Minute)
	refreshLimiter := middleware.NewTokenBucketLimiter(30, time.Minute)
	logoutLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	ratingLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)

	authGroup := r.Group("")
	{
		authGroup.POST("/register", registerLimiter.Limit, h.Register.Register)
		authGroup.POST("/verify-registration", verifyRegistrationLimiter.Limit, h.Register.VerifyRegistration)
		authGroup.POST("/guest-login", guestLoginLimiter.Limit, h.Register.GuestLogin)
		authGroup.POST("/login", loginLimiter.Limit, h.Login.Login)
		authGroup.POST("/refresh", refreshLimiter.Limit, h.Refresh.Refresh)

		authGroup.POST("/logout", logoutLimiter.Limit, h.Logout.Logout)
		authGroup.POST("/logout-all", logoutLimiter.Limit, h.Logout.LogoutAll)

		authGroup.POST("/forgot-password", forgotLimiter.Limit, h.PasswordReset.ForgotPassword)
		authGroup.POST("/verify-reset-code", verifyResetLimiter.Limit, h.PasswordReset.VerifyCode)
		authGroup.POST("/reset-password", resetPasswordLimiter.Limit, h.PasswordReset.ResetPassword)
	}

	protected.POST("/rating", ratingLimiter.Limit, middleware.RequireRegisteredUser(), h.Rating.SetRating)
}
