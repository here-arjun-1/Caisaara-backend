package service

import (
	"errors"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	UserRepository    *repository.UserRepository
	SessionRepository *repository.SessionRepository
}

func NewLoginService(
	userRepository *repository.UserRepository,
	sessionRepository *repository.SessionRepository,
) *LoginService {
	return &LoginService{
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
	}
}

func (h *LoginService) Login(
	req dto.LoginData,
) (string, string, error) {

	if req.Username == "" {
		return "", "", errors.New("username is required")
	}
	if req.Password == "" {
		return "", "", errors.New("password is required")
	}
	user, err := h.UserRepository.FindUserByUsername(req.Username)
	if err != nil {
		return "", "", errors.New("invalid username and password")
	}
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)
	if err != nil {
		return "", "", errors.New("invalid username and password")
	}
	accessToken, err := token.GenerateAccessToken(user.ID)
	if err != nil {
		return "", "", errors.New("failed to generate access token")
	}
	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		return "", "", errors.New("failed to generate refresh token")
	}

	refreshTokenHash := token.HashRefreshToken(refreshToken)

	session := &model.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(30 * 24 * time.Hour),
	}
	err = h.SessionRepository.CreateSession(session)
	if err != nil {
		return "", "", errors.New("failed to create session")
	}
	return accessToken, refreshToken, nil
}
