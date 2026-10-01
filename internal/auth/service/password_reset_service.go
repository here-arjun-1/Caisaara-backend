package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const maxOTPAttempts = 5

type OTPData struct {
	Hash     string `json:"hash"`
	Attempts int    `json:"attempts"`
}

type PasswordResetService struct {
	UserRepository          *repository.UserRepository
	PasswordResetRepository *repository.PasswordResetRepository
	SessionRepository       *repository.SessionRepository
	RedisClient             *redis.Client
}

func NewPasswordResetService(
	userRepository *repository.UserRepository,
	passwordResetRepository *repository.PasswordResetRepository,
	sessionRepository *repository.SessionRepository,
	redisClient *redis.Client,
) *PasswordResetService {
	return &PasswordResetService{
		UserRepository:          userRepository,
		PasswordResetRepository: passwordResetRepository,
		SessionRepository:       sessionRepository,
		RedisClient:             redisClient,
	}
}

func (s *PasswordResetService) ForgotPassword(req dto.ForgotPasswordRequest) error {
	userEmail := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.UserRepository.FindUserByEmail(userEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		log.Printf("find user by email failed: %v", err)
		return ErrInternal
	}

	go s.sendResetCode(user.Email)

	return nil
}

func (s *PasswordResetService) sendResetCode(userEmail string) {
	otp, err := generateOTP()
	if err != nil {
		log.Printf("generate otp failed: %v", err)
		return
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash otp failed: %v", err)
		return
	}

	data := OTPData{
		Hash:     string(otpHash),
		Attempts: 0,
	}
	dataJSON, _ := json.Marshal(data)

	if err := s.RedisClient.Set(context.Background(), "forgot_otp:"+userEmail, dataJSON, 15*time.Minute).Err(); err != nil {
		log.Printf("save otp to redis failed: %v", err)
		return
	}

	if err := email.SendPasswordResetEmail(userEmail, otp); err != nil {
		log.Printf("send password reset email failed: %v", err)
	}
}

func (s *PasswordResetService) VerifyCode(req dto.VerifyCodeRequest) (string, error) {
	userEmail := strings.ToLower(strings.TrimSpace(req.Email))
	ctx := context.Background()
	key := "forgot_otp:" + userEmail

	var resetToken string
	var returnErr error

	err := s.RedisClient.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err != nil {
			returnErr = errors.New("invalid or expired verification code")
			return nil
		}

		var data OTPData
		if err := json.Unmarshal([]byte(val), &data); err != nil {
			returnErr = errors.New("invalid or expired verification code")
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

		if err := bcrypt.CompareHashAndPassword([]byte(data.Hash), []byte(req.Code)); err != nil {
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, newData, ttl)
				return nil
			})
			returnErr = errors.New("invalid verification code")
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

func (s *PasswordResetService) ResetPassword(req dto.ResetPasswordRequest) error {
	ctx := context.Background()
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
		log.Printf("hash new password failed: %v", err)
		return ErrInternal
	}

	err = s.PasswordResetRepository.ResetPassword(email, string(hashedPassword))
	if err != nil {
		log.Printf("reset password failed: %v", err)
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
