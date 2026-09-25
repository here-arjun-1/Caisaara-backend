package service

import (
	"errors"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	UserRepository *repository.UserRepository
}

func NewLoginService(userRepository *repository.UserRepository) *LoginService {
	return &LoginService{
		UserRepository: userRepository,
	}
}

func (h *LoginService) Login(req dto.LoginData) (string, error) {
	if req.Username == "" {
		return "", errors.New("username is required")
	}
	if req.Password == "" {
		return "", errors.New("password is required")
	}

	user, err := h.UserRepository.FindUserByUsername(req.Username)

	if err != nil {
		return "", errors.New("invalid username and password")
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)

	if err != nil {
		return "", errors.New("invalid username and password")
	}

	accessToken, err := token.GenerateAccessToken(user.ID)
	if err != nil {
		return "", errors.New("failed to generate access token")
	}

	return accessToken, nil
}
