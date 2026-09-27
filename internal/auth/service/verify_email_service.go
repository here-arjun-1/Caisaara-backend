package service

import (
	"errors"
	"log"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/jackc/pgx/v5"
)

type VerifyEmailService struct {
	UserRepository *repository.UserRepository
}

func NewVerifyEmailService(
	userRepository *repository.UserRepository,
) *VerifyEmailService {

	return &VerifyEmailService{
		UserRepository: userRepository,
	}
}

func (s *VerifyEmailService) VerifyEmail(
	token string,
) error {

	if token == "" {
		return errors.New("verification token is required")
	}

	err := s.UserRepository.VerifyEmail(token)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("invalid or expired verification token")
	}
	if err != nil {
		log.Printf("verify email failed: %v", err)
		return ErrInternal
	}

	return nil
}
