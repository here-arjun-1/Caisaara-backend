package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	pendingRegistrations sync.Map
	ErrInvalidCode       = errors.New("invalid or expired verification code")
)

type pendingUser struct {
	Req       dto.RegisterData
	Code      string
	ExpiresAt time.Time
}

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

func (s *RegisterService) Register(req dto.RegisterData) error {
	if !isValidUsername(req.Username) {
		return ErrInvalidUsername
	}

	req.Email = strings.ToLower(req.Email)

	_, err := s.UserRepository.FindUserByEmail(req.Email)
	if err == nil {
		return ErrEmailTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("find user by email failed: %v", err)
		return ErrInternal
	}

	_, err = s.UserRepository.FindUserByUsername(req.Username)
	if err == nil {
		return ErrUsernameTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("find user by username failed: %v", err)
		return ErrInternal
	}

	code, err := generateOTP()
	if err != nil {
		log.Printf("generate otp failed: %v", err)
		return ErrInternal
	}

	pendingRegistrations.Store(req.Email, pendingUser{
		Req:       req,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	})

	go email.SendRegistrationEmail(req.Email, code)

	return nil
}

func (s *RegisterService) VerifyRegistration(req dto.VerifyRegistrationData) (string, string, error) {
	req.Email = strings.ToLower(req.Email)
	val, ok := pendingRegistrations.Load(req.Email)
	if !ok {
		return "", "", ErrInvalidCode
	}

	pUser := val.(pendingUser)
	if time.Now().After(pUser.ExpiresAt) || pUser.Code != req.Code {
		return "", "", ErrInvalidCode
	}

	pendingRegistrations.Delete(req.Email)

	userReq := pUser.Req

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(userReq.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		log.Printf("hash password failed: %v", err)
		return "", "", ErrInternal
	}

	user := &model.User{
		Username: userReq.Username,
		Email:    userReq.Email,
		Password: string(hashedPassword),
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

		return "", "", ErrInternal
	}

	return createSessionTokens(s.SessionRepository, user.ID)
}

func (s *RegisterService) GuestLogin(req dto.GuestLoginData) (string, string, error) {
	if !isValidUsername(req.Username) {
		return "", "", ErrInvalidUsername
	}

	_, err := s.UserRepository.FindUserByUsername(req.Username)
	if err == nil {
		return "", "", ErrUsernameTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("find user by username failed: %v", err)
		return "", "", ErrInternal
	}

	dummyEmail := fmt.Sprintf("%s_%d@guest.local", req.Username, time.Now().UnixNano())

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("guest_password_dummy"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash password failed: %v", err)
		return "", "", ErrInternal
	}

	user := &model.User{
		Username: req.Username,
		Email:    dummyEmail,
		Password: string(hashedPassword),
	}

	err = s.UserRepository.CreateUser(user)
	if err != nil {
		log.Printf("create user failed: %v", err)
		return "", "", ErrInternal
	}

	return createSessionTokens(s.SessionRepository, user.ID)
}

func isValidUsername(username string) bool {
	for _, char := range username {
		isLetter := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
		isNumber := char >= '0' && char <= '9'
		if !isLetter && !isNumber && char != '_' {
			return false
		}
	}
	return true
}
