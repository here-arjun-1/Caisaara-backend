package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
)

func createSessionTokens(
	ctx context.Context,
	sessionRepository SessionRepository,
	jwtSecret string,
	userID int64,
	passwordVersion int,
) (string, string, error) {

	accessToken, err := token.GenerateAccessToken(jwtSecret, userID, passwordVersion)
	if err != nil {
		slog.Error("generate access token failed", "error", err)
		return "", "", ErrInternal
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		slog.Error("generate refresh token failed", "error", err)
		return "", "", ErrInternal
	}

	refreshTokenHash := token.HashRefreshToken(refreshToken)

	session := &model.Session{
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	err = sessionRepository.CreateSession(ctx, session)
	if err != nil {
		slog.Error("create session failed", "error", err)
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}

func createSessionTokensTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionRepository SessionRepository,
	jwtSecret string,
	userID int64,
	passwordVersion int,
) (string, string, error) {

	accessToken, err := token.GenerateAccessToken(jwtSecret, userID, passwordVersion)
	if err != nil {
		slog.Error("generate access token failed", "error", err)
		return "", "", ErrInternal
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		slog.Error("generate refresh token failed", "error", err)
		return "", "", ErrInternal
	}

	refreshTokenHash := token.HashRefreshToken(refreshToken)

	session := &model.Session{
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	err = sessionRepository.CreateSessionTx(ctx, tx, session)
	if err != nil {
		slog.Error("create session failed", "error", err)
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}
