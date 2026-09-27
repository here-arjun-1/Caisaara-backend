package service

import (
	"errors"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

func createSessionTokens(
	sessionRepository *repository.SessionRepository,
	userID int64,
) (string, string, error) {

	accessToken, err := token.GenerateAccessToken(userID)
	if err != nil {
		return "", "", errors.New("failed to generate access token")
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		return "", "", errors.New("failed to generate refresh token")
	}

	refreshTokenHash := token.HashRefreshToken(refreshToken)

	session := &model.Session{
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}

	err = sessionRepository.CreateSession(session)
	if err != nil {
		return "", "", errors.New("failed to create session")
	}

	return accessToken, refreshToken, nil
}
