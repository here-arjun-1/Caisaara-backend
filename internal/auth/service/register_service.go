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

func ValidatePassword(password string) error {

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	var hasUpper bool
	var hasLower bool
	var hasNumber bool
	var hasSpecial bool

	for _, char := range password {

		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true

		case char >= 'a' && char <= 'z':
			hasLower = true

		case char >= '0' && char <= '9':
			hasNumber = true

		case char == '!' || char == '@' || char == '#' ||
			char == '$' || char == '%' || char == '^' ||
			char == '&' || char == '*':
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain an uppercase letter")
	}

	if !hasLower {
		return errors.New("password must contain a lowercase letter")
	}

	if !hasNumber {
		return errors.New("password must contain a number")
	}

	if !hasSpecial {
		return errors.New("password must contain a special character")
	}

	return nil
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

	err = ValidatePassword(req.Password)

	if err != nil {
		return err
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
