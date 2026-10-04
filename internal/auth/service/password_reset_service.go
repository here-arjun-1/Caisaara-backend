package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const maxOTPAttempts = 5

type OTPData struct {
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
}

type PasswordResetService struct {
	UserRepository          UserRepository
	PasswordResetRepository PasswordResetRepository
	SessionRepository       SessionRepository
	RedisClient             *redis.Client
	TaskDistributor         *asynq.Client
}

func NewPasswordResetService(
	userRepository UserRepository,
	passwordResetRepository PasswordResetRepository,
	sessionRepository SessionRepository,
	redisClient *redis.Client,
	taskDistributor *asynq.Client,
) *PasswordResetService {
	return &PasswordResetService{
		UserRepository:          userRepository,
		PasswordResetRepository: passwordResetRepository,
		SessionRepository:       sessionRepository,
		RedisClient:             redisClient,
		TaskDistributor:         taskDistributor,
	}
}

func (s *PasswordResetService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	userEmail := strings.ToLower(strings.TrimSpace(req.Email))

	_, err := s.UserRepository.FindUserByEmail(ctx, userEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		slog.Error("find user by email failed", "error", err)
		return ErrInternal
	}

	otp, err := generateOTP()
	if err != nil {
		slog.Error("generate otp failed", "error", err)
		return ErrInternal
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash otp failed", "error", err)
		return ErrInternal
	}

	data := OTPData{
		CodeHash:  string(otpHash),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	dataJSON, _ := json.Marshal(data)

	if err := s.RedisClient.Set(ctx, "forgot_otp:"+userEmail, dataJSON, 15*time.Minute).Err(); err != nil {
		slog.Error("save otp to redis failed", "error", err)
		return ErrInternal
	}
	s.RedisClient.Set(ctx, "raw_reset_otp:"+userEmail, otp, 15*time.Minute)
	slog.Info("generated password reset OTP", "email", userEmail, "otp", otp)

	task, err := worker.NewEmailPasswordResetTask(userEmail, otp)
	if err != nil {
		slog.Error("could not create password reset email task", "error", err)
	} else {
		info, err := s.TaskDistributor.EnqueueContext(ctx, task)
		if err != nil {
			slog.Error("could not enqueue password reset email task", "error", err)
		} else {
			slog.Info("enqueued password reset email task", "id", info.ID, "queue", info.Queue)
		}
	}

	return nil
}

func (s *PasswordResetService) VerifyCode(ctx context.Context, req dto.VerifyCodeRequest) (string, error) {
	userEmail := strings.ToLower(strings.TrimSpace(req.Email))
	key := "forgot_otp:" + userEmail

	var resetToken string
	var returnErr error

	err := s.RedisClient.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err != nil {
			returnErr = ErrInvalidCode
			return nil
		}

		var data OTPData
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			returnErr = ErrInvalidCode
			return nil
		}

		if time.Now().After(data.ExpiresAt) {
			returnErr = ErrInvalidCode
			return nil
		}

		data.Attempts++
		if data.Attempts > maxOTPAttempts {
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, key)
				return nil
			})
			returnErr = errors.New("too many attempts, please request a new code")
			return err
		}

		newData, _ := json.Marshal(data)
		ttl := tx.TTL(ctx, key).Val()

		if err := bcrypt.CompareHashAndPassword([]byte(data.CodeHash), []byte(req.Code)); err != nil {
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, newData, ttl)
				return nil
			})
			returnErr = ErrInvalidCode
			return err
		}

		resetToken, err = generateResetToken()
		if err != nil {
			returnErr = ErrInternal
			return nil
		}

		resetTokenHash := hashResetToken(resetToken)

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, key)
			pipe.Set(ctx, "reset_token:"+resetTokenHash, userEmail, 15*time.Minute)
			return nil
		})
		return err
	}, key)

	if err != nil {
		if err == redis.TxFailedErr {
			return "", errors.New("concurrent request, please try again")
		}
		return "", ErrInternal
	}
	if returnErr != nil {
		return "", returnErr
	}

	return resetToken, nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	key := "reset_token:" + hashResetToken(req.ResetToken)

	email, err := s.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	deleted, err := s.RedisClient.Del(ctx, key).Result()
	if err != nil || deleted == 0 {
		return errors.New("invalid or expired reset token")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash new password failed", "error", err)
		return ErrInternal
	}

	err = s.PasswordResetRepository.ResetPassword(ctx, email, string(hashedPassword))
	if err != nil {
		slog.Error("reset password failed", "error", err)
		return ErrInternal
	}

	return nil
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashResetToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
