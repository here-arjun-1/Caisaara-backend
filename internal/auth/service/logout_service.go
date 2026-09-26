package service

import (
	"errors"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

type LogoutService struct {
	SessionRepository *repository.SessionRepository
}

func NewLogoutService(sessionRepository *repository.SessionRepository) *LogoutService {
	return &LogoutService{
		SessionRepository: sessionRepository,
	}
}

func (h *LogoutService) Logout(req dto.LogoutData) error {
	if req.RefreshToken == "" {
		return errors.New("refresh token is required")
	}
	refreshTokenHash := token.HashRefreshToken(req.RefreshToken)

	err := h.SessionRepository.RevokeSession(refreshTokenHash)

	if err != nil {
		return errors.New("failed to logout")
	}
	return nil
}
