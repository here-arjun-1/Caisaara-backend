package service

import (
	"errors"
	"log"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/jackc/pgx/v5"
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
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrInvalidCredentials
	}
	if err != nil {
		log.Printf("find user failed: %v", err)
		return "", "", ErrInternal
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)

	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	return createSessionTokens(h.SessionRepository, user.ID)
}
