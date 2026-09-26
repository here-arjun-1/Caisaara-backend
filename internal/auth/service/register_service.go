package service

import (
	"errors"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterService struct {
	UserRepository *repository.UserRepository
}

func NewRegisterService(userRepository *repository.UserRepository) *RegisterService {
	return &RegisterService{
		UserRepository: userRepository,
	}
}

func (s *RegisterService) Register(req dto.RegisterData) error {

	if req.Username == "" {
		return errors.New("username is required")
	}
	existingUser, err := s.UserRepository.FindByUsername(req.Username)

	if err == nil && existingUser != nil {
		return errors.New("username already exists")
	}

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return errors.New("failed to check username")
	}

	if req.Password == "" {
		return errors.New("password is required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return errors.New("failed to process password")
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
	}

	err = s.UserRepository.CreateUser(user)

	if err != nil {
		return errors.New("failed to create user")
	}

	return nil
}
