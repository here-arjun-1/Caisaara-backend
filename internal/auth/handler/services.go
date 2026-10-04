package handler

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
)

type LoginService interface {
	Login(ctx context.Context, req dto.LoginData) (string, string, string, bool, error)
}

type LogoutService interface {
	Logout(ctx context.Context, req dto.LogoutData) error
	LogoutAll(ctx context.Context, req dto.LogoutData) error
}

type RefreshService interface {
	Refresh(ctx context.Context, req dto.RefreshData) (string, string, error)
}

type RegisterService interface {
	Register(ctx context.Context, req dto.RegisterData) error
	VerifyRegistration(ctx context.Context, req dto.VerifyRegistrationData) (string, string, error)
	GuestLogin(ctx context.Context) (string, string, error)
}

type PasswordResetService interface {
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error
	VerifyCode(ctx context.Context, req dto.VerifyCodeRequest) (string, error)
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
}

type RatingService interface {
	SetInitialRating(ctx context.Context, userID int64, level string) (int, error)
}
