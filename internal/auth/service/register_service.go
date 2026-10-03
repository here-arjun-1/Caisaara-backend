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
	CodeHash  string
	Attempts  int
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

func (s *RegisterService) Register(ctx context.Context, req dto.RegisterData) error {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))

	_, err := s.UserRepository.FindUserByEmail(ctx, req.Email)
	if err == nil {
		return ErrEmailTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("find user by email failed: %v", err)
		return ErrInternal
	}

	_, err = s.UserRepository.FindUserByUsername(ctx, req.Username)
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

	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash otp failed: %v", err)
		return ErrInternal
	}

	pUser := pendingUser{
		Req:       req,
		CodeHash:  string(codeHash),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	pUserJSON, _ := json.Marshal(pUser)
	s.RedisClient.Set(ctx, "register:"+req.Email, pUserJSON, 15*time.Minute)

	go func() {
		if err := email.SendRegistrationEmail(req.Email, code); err != nil {
			log.Printf("send registration email failed: %v", err)
		}
	}()

	return nil
}

func (s *RegisterService) VerifyRegistration(ctx context.Context, req dto.VerifyRegistrationData) (string, string, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	key := "register:" + req.Email

	var pUser pendingUser
	var returnErr error

	err := s.RedisClient.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err != nil {
			returnErr = ErrInvalidCode
			return nil
		}

		if err := json.Unmarshal([]byte(val), &pUser); err != nil {
			returnErr = ErrInvalidCode
			return nil
		}

		if time.Now().After(pUser.ExpiresAt) {
			returnErr = ErrInvalidCode
			return nil
		}

		pUser.Attempts++
		if pUser.Attempts > maxOTPAttempts {
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, key)
				return nil
			})
			returnErr = errors.New("too many attempts, please request a new code")
			return err
		}

		newData, _ := json.Marshal(pUser)
		ttl := tx.TTL(ctx, key).Val()

		if err := bcrypt.CompareHashAndPassword([]byte(pUser.CodeHash), []byte(req.Code)); err != nil {
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, newData, ttl)
				return nil
			})
			returnErr = ErrInvalidCode
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			return nil
		})
		return err
	}, key)

	if err != nil {
		if errors.Is(err, redis.TxFailedErr) {
			return "", "", errors.New("concurrent request, please try again")
		}
		return "", "", ErrInternal
	}
	if returnErr != nil {
		return "", "", returnErr
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

	err = s.UserRepository.CreateUser(ctx, user)

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

	return createSessionTokens(ctx, s.SessionRepository, user.ID)
}

func (s *RegisterService) GuestLogin(ctx context.Context) (string, error) {
	guestID, err := token.GenerateGuestID()
	if err != nil {
		log.Printf("generate guest id failed: %v", err)
		return "", ErrInternal
	}

	return guestID, nil
}
