package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
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

func (h *RefreshService) Refresh(ctx context.Context, req dto.RefreshData) (string, string, error) {

	if req.RefreshToken == "" {
		return "", "", errors.New("refresh token is required")
	}

	oldRefreshTokenHash := token.HashRefreshToken(
		req.RefreshToken,
	)

	session, err := h.SessionRepository.FindSessionByRefreshTokenHash(
		ctx,
		oldRefreshTokenHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", errors.New("invalid refresh token")
	}
	if err != nil {
		log.Printf("find session failed: %v", err)
		return "", "", ErrInternal
	}

	if session.RevokedAt != nil {
		err = h.SessionRepository.RevokeAllSessions(ctx, session.UserID)
		if err != nil {
			log.Printf("revoke all sessions failed: %v", err)
		}
		return "", "", errors.New("refresh token has been revoked")
	}

	if time.Now().After(session.ExpiresAt) {
		return "", "", errors.New("refresh token has expired")
	}

	newRefreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		log.Printf("generate refresh token failed: %v", err)
		return "", "", ErrInternal
	}

	newSession := &model.Session{
		RefreshTokenHash: token.HashRefreshToken(newRefreshToken),
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	err = h.SessionRepository.RotateSession(
		ctx,
		oldRefreshTokenHash,
		newSession,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", errors.New("refresh token has been revoked")
	}
	if err != nil {
		log.Printf("rotate session failed: %v", err)
		return "", "", ErrInternal
	}

	accessToken, err := token.GenerateAccessToken(
		newSession.UserID,
	)

	if err != nil {
		log.Printf("generate access token failed: %v", err)
		return "", "", ErrInternal
	}

	return accessToken, newRefreshToken, nil
}
