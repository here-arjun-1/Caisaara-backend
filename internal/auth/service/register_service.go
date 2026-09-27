package service

import (
	"errors"
	"log"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken = errors.New("username already taken")
	ErrEmailTaken    = errors.New("email already registered")
)

type RegisterService struct {
	UserRepository    *repository.UserRepository
	SessionRepository *repository.SessionRepository
}

func NewRegisterService(
	userRepository *repository.UserRepository,
	sessionRepository *repository.SessionRepository,
) *RegisterService {

	return &RegisterService{
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
	}
}

func (s *RegisterService) Register(
	req dto.RegisterData,
) (string, string, error) {

	if req.Username == "" {
		return "", "", errors.New("username is required")
	}

	if req.Email == "" {
		return "", "", errors.New("email is required")
	}

	if req.Password == "" {
		return "", "", errors.New("password is required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return "", "", errors.New("failed to process password")
	}

	verificationToken, err := token.GenerateEmailVerificationToken()

	if err != nil {
		return "", "", errors.New("failed to generate verification token")
	}

	user := &model.User{
		Username:                   req.Username,
		Email:                      req.Email,
		Password:                   string(hashedPassword),
		EmailVerified:              false,
		EmailVerificationToken:     verificationToken,
		EmailVerificationExpiresAt: time.Now().Add(15 * time.Minute),
	}

	err = s.UserRepository.CreateUser(user)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return "", "", ErrUsernameTaken
			case "users_email_key":
				return "", "", ErrEmailTaken
			}
		}

		log.Printf("create user failed: %v", err)

		return "", "", errors.New("failed to create user")
	}

	verificationLink :=
		"http://localhost:8050/verify-email?token=" +
			verificationToken

	err = email.SendVerificationEmail(
		req.Email,
		verificationLink,
	)

	if err != nil {
		log.Printf("send verification email failed: %v", err)
	}

	return createSessionTokens(s.SessionRepository, user.ID)
}
