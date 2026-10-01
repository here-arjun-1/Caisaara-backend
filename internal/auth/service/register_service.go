package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCode = errors.New("invalid or expired verification code")
)

type pendingUser struct {
	Req       dto.RegisterData
	Code      string
	ExpiresAt time.Time
}

type RegisterService struct {
	UserRepository    *repository.UserRepository
	SessionRepository *repository.SessionRepository
	RedisClient       *redis.Client
}

func NewRegisterService(
	userRepository *repository.UserRepository,
	sessionRepository *repository.SessionRepository,
	redisClient *redis.Client,
) *RegisterService {

	return &RegisterService{
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		RedisClient:       redisClient,
	}
}

func (s *RegisterService) Register(req dto.RegisterData) error {
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

	pUser := pendingUser{
		Req:       req,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	pUserJSON, _ := json.Marshal(pUser)
	s.RedisClient.Set(context.Background(), "register:"+req.Email, pUserJSON, 15*time.Minute)

	go func() {
		if err := email.SendRegistrationEmail(req.Email, code); err != nil {
			log.Printf("send registration email failed: %v", err)
		}
	}()

	return nil
}

func (s *RegisterService) VerifyRegistration(req dto.VerifyRegistrationData) (string, string, error) {
	req.Email = strings.ToLower(req.Email)
	val, err := s.RedisClient.Get(context.Background(), "register:"+req.Email).Result()
	if err != nil {
		return "", "", ErrInvalidCode
	}

	var pUser pendingUser
	if err := json.Unmarshal([]byte(val), &pUser); err != nil {
		return "", "", ErrInvalidCode
	}

	if time.Now().After(pUser.ExpiresAt) || pUser.Code != req.Code {
		return "", "", ErrInvalidCode
	}

	deleted, err := s.RedisClient.Del(context.Background(), "register:"+req.Email).Result()
	if err != nil || deleted == 0 {
		return "", "", ErrInvalidCode
	}

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

func (s *RegisterService) GuestLogin(username string) (string, string, error) {
	guestID, err := token.GenerateGuestID()
	if err != nil {
		log.Printf("generate guest id failed: %v", err)
		return "", "", ErrInternal
	}

	accessToken, err := token.GenerateGuestToken(guestID, username)
	if err != nil {
		log.Printf("generate guest token failed: %v", err)
		return "", "", ErrInternal
	}

	return guestID, accessToken, nil
}
