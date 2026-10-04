package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
)

type LogoutService struct {
	SessionRepository SessionRepository
}

func NewLogoutService(
	sessionRepository SessionRepository,
) *LogoutService {

	return &LogoutService{
		SessionRepository: sessionRepository,
	}
}

func (s *LogoutService) Logout(
	ctx context.Context,
	req dto.LogoutData,
) error {

	if req.RefreshToken == "" {
		return errors.New("refresh token is required")
	}

	refreshTokenHash := token.HashRefreshToken(
		req.RefreshToken,
	)

	err := s.SessionRepository.RevokeSession(
		ctx,
		refreshTokenHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New(
			"invalid or already revoked refresh token",
		)
	}

	if err != nil {
		slog.Error("revoke session failed", "error", err)
		return ErrInternal
	}

	return nil
}

func (s *LogoutService) LogoutAll(
	ctx context.Context,
	req dto.LogoutData,
) error {

	if req.RefreshToken == "" {
		return errors.New("refresh token is required")
	}

	refreshTokenHash := token.HashRefreshToken(
		req.RefreshToken,
	)

	err := s.SessionRepository.RevokeAllSessionsByTokenHash(
		ctx,
		refreshTokenHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("invalid refresh token")
	}
	if err != nil {
		if err.Error() == "refresh token has been revoked" {
			return err
		}
		slog.Error("revoke all sessions failed", "error", err)
		return ErrInternal
	}

	return nil
}
