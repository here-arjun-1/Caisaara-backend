package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
	DB                *pgxpool.Pool
	UserRepository    *repository.UserRepository
	SessionRepository *repository.SessionRepository
	RedisClient       *redis.Client
	TaskDistributor   *asynq.Client
	JWTSecret         string
}

func NewRegisterService(
	db *pgxpool.Pool,
	userRepository *repository.UserRepository,
	sessionRepository *repository.SessionRepository,
	redisClient *redis.Client,
	taskDistributor *asynq.Client,
	jwtSecret string,
) *RegisterService {

	return &RegisterService{
		DB:                db,
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		RedisClient:       redisClient,
		TaskDistributor:   taskDistributor,
		JWTSecret:         jwtSecret,
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash password failed: %v", err)
		return ErrInternal
	}
	req.Password = string(hashedPassword)

	pUser := pendingUser{
		Req:       req,
		CodeHash:  string(codeHash),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	pUserJSON, _ := json.Marshal(pUser)
	s.RedisClient.Set(ctx, "register:"+req.Email, pUserJSON, 15*time.Minute)

	task, err := worker.NewEmailRegistrationTask(req.Email, code)
	if err != nil {
		log.Printf("could not create email task: %v", err)
	} else {
		info, err := s.TaskDistributor.EnqueueContext(ctx, task)
		if err != nil {
			log.Printf("could not enqueue email task: %v", err)
		} else {
			log.Printf("enqueued email task: id=%s queue=%s", info.ID, info.Queue)
		}
	}

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
		if err == redis.TxFailedErr {
			return "", "", errors.New("concurrent request, please try again")
		}
		return "", "", ErrInternal
	}
	if returnErr != nil {
		return "", "", returnErr
	}

	userReq := pUser.Req

	user := &model.User{
		Username: userReq.Username,
		Email:    userReq.Email,
		Password: userReq.Password,
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		log.Printf("begin tx failed: %v", err)
		return "", "", ErrInternal
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	err = s.UserRepository.CreateUserTx(ctx, tx, user)

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

	accessToken, refreshToken, err := createSessionTokensTx(ctx, tx, s.SessionRepository, s.JWTSecret, user.ID)

	if err != nil {
		return "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("commit tx failed: %v", err)
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}

func (s *RegisterService) GuestLogin() (string, error) {
	guestID, err := token.GenerateGuestID()
	if err != nil {
		log.Printf("generate guest id failed: %v", err)
		return "", ErrInternal
	}

	return guestID, nil
}
