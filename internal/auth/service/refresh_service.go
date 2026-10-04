package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
)

type RefreshService struct {
	SessionRepository SessionRepository
	UserRepository    UserRepository
	JWTSecret         string
}

func NewRefreshService(
	sessionRepository SessionRepository,
	userRepository UserRepository,
	jwtSecret string,
) *RefreshService {
	return &RefreshService{
		SessionRepository: sessionRepository,
		UserRepository:    userRepository,
		JWTSecret:         jwtSecret,
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
		slog.Error("find session failed", "error", err)
		return "", "", ErrInternal
	}

	if session.RevokedAt != nil {
		err = h.SessionRepository.RevokeAllSessions(ctx, session.UserID)
		if err != nil {
			slog.Error("revoke all sessions failed", "error", err)
		}
		return "", "", errors.New("refresh token has been revoked")
	}

	if time.Now().After(session.ExpiresAt) {
		return "", "", errors.New("refresh token has expired")
	}

	newRefreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		slog.Error("generate refresh token failed", "error", err)
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
		slog.Error("rotate session failed", "error", err)
		return "", "", ErrInternal
	}

	user, err := h.UserRepository.FindUserByID(ctx, newSession.UserID)
	if err != nil {
		slog.Error("find user by id failed", "error", err)
		return "", "", ErrInternal
	}

	accessToken, err := token.GenerateAccessToken(
		h.JWTSecret,
		newSession.UserID,
		user.PasswordVersion,
	)

	if err != nil {
		slog.Error("generate access token failed", "error", err)
		return "", "", ErrInternal
	}

	return accessToken, newRefreshToken, nil
}
