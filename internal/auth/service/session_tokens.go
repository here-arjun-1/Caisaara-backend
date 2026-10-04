package service

import (
	"context"
	"log"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
)

func createSessionTokens(
	ctx context.Context,
	sessionRepository *repository.SessionRepository,
	jwtSecret string,
	userID int64,
) (string, string, error) {

	accessToken, err := token.GenerateAccessToken(jwtSecret, userID)
	if err != nil {
		log.Printf("generate access token failed: %v", err)
		return "", "", ErrInternal
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		log.Printf("generate refresh token failed: %v", err)
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
		log.Printf("create session failed: %v", err)
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}

func createSessionTokensTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionRepository *repository.SessionRepository,
	jwtSecret string,
	userID int64,
) (string, string, error) {

	accessToken, err := token.GenerateAccessToken(jwtSecret, userID)
	if err != nil {
		log.Printf("generate access token failed: %v", err)
		return "", "", ErrInternal
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		log.Printf("generate refresh token failed: %v", err)
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
		log.Printf("create session failed: %v", err)
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}
