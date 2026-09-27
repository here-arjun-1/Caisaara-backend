package service

import (
	"errors"
	"log"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
)

type RefreshService struct {
	SessionRepository *repository.SessionRepository
}

func NewRefreshService(
	sessionRepository *repository.SessionRepository,
) *RefreshService {
	return &RefreshService{
		SessionRepository: sessionRepository,
	}
}

func (h *RefreshService) Refresh(req dto.RefreshData) (string, error) {

	if req.RefreshToken == "" {
		return "", errors.New("refresh token is required")
	}

	refreshTokenHash := token.HashRefreshToken(
		req.RefreshToken,
	)

	session, err := h.SessionRepository.FindSessionByRefreshTokenHash(
		refreshTokenHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("invalid refresh token")
	}
	if err != nil {
		log.Printf("find session failed: %v", err)
		return "", ErrInternal
	}

	if session.RevokedAt != nil {
		return "", errors.New("refresh token has been revoked")
	}

	if time.Now().After(session.ExpiresAt) {
		return "", errors.New("refresh token has expired")
	}

	accessToken, err := token.GenerateAccessToken(
		session.UserID,
	)

	if err != nil {
		log.Printf("generate access token failed: %v", err)
		return "", ErrInternal
	}

	return accessToken, nil
}
