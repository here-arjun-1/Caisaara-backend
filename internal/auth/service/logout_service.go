package service

import (
	"errors"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
)

type LogoutService struct {
	SessionRepository *repository.SessionRepository
}

func NewLogoutService(
	sessionRepository *repository.SessionRepository,
) *LogoutService {

	return &LogoutService{
		SessionRepository: sessionRepository,
	}
}

func (s *LogoutService) Logout(
	req dto.LogoutData,
) error {

	if req.RefreshToken == "" {
		return errors.New("refresh token is required")
	}

	refreshTokenHash := token.HashRefreshToken(
		req.RefreshToken,
	)

	err := s.SessionRepository.RevokeSession(
		refreshTokenHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New(
			"invalid or already revoked refresh token",
		)
	}

	if err != nil {
		return errors.New("failed to logout")
	}

	return nil
}

func (s *LogoutService) LogoutAll(
	req dto.LogoutData,
) error {

	if req.RefreshToken == "" {
		return errors.New("refresh token is required")
	}

	refreshTokenHash := token.HashRefreshToken(
		req.RefreshToken,
	)

	session, err := s.SessionRepository.FindSessionByRefreshTokenHash(
		refreshTokenHash,
	)

	if err != nil {
		return errors.New("invalid refresh token")
	}

	if session.RevokedAt != nil {
		return errors.New("refresh token has been revoked")
	}

	err = s.SessionRepository.RevokeAllSessions(
		session.UserID,
	)

	if err != nil {
		return errors.New("failed to logout all sessions")
	}

	return nil
}
