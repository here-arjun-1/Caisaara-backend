package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	"golang.org/x/crypto/bcrypt"
)

const maxOTPAttempts = 5

type PasswordResetService struct {
	UserRepository          *repository.UserRepository
	PasswordResetRepository *repository.PasswordResetRepository
	SessionRepository       *repository.SessionRepository
}

func NewPasswordResetService(
	userRepository *repository.UserRepository,
	passwordResetRepository *repository.PasswordResetRepository,
	sessionRepository *repository.SessionRepository,
) *PasswordResetService {
	return &PasswordResetService{
		UserRepository:          userRepository,
		PasswordResetRepository: passwordResetRepository,
		SessionRepository:       sessionRepository,
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

	expiresAt := time.Now().Add(15 * time.Minute)
	if err := s.PasswordResetRepository.SaveOTP(userEmail, string(otpHash), expiresAt); err != nil {
		log.Printf("save otp failed: %v", err)
		return
	}

	if err := email.SendPasswordResetEmail(userEmail, otp); err != nil {
		log.Printf("send password reset email failed: %v", err)
	}
}

func (s *PasswordResetService) VerifyCode(req dto.VerifyCodeRequest) (string, error) {
	userEmail := strings.ToLower(strings.TrimSpace(req.Email))

	pr, err := s.PasswordResetRepository.FindByEmail(userEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("invalid or expired verification code")
	}
	if err != nil {
		log.Printf("find password reset failed: %v", err)
		return "", ErrInternal
	}

	if time.Now().After(pr.OTPExpiresAt) {
		return "", errors.New("verification code expired")
	}

	attempts, err := s.PasswordResetRepository.IncrementAttempts(pr.Email)
	if err != nil {
		log.Printf("increment otp attempts failed: %v", err)
		return "", ErrInternal
	}
	if attempts > maxOTPAttempts {
		return "", errors.New("too many attempts, please request a new code")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pr.OTPHash), []byte(req.Code)); err != nil {
		return "", errors.New("invalid verification code")
	}

	resetToken, err := generateResetToken()
	if err != nil {
		log.Printf("generate reset token failed: %v", err)
		return "", ErrInternal
	}

	resetTokenHash := hashResetToken(resetToken)
	expiresAt := time.Now().Add(15 * time.Minute)

	err = s.PasswordResetRepository.SaveResetToken(pr.Email, resetTokenHash, expiresAt)
	if err != nil {
		log.Printf("save reset token failed: %v", err)
		return "", ErrInternal
	}

	return resetToken, nil
}

func (s *PasswordResetService) ResetPassword(req dto.ResetPasswordRequest) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash new password failed: %v", err)
		return ErrInternal
	}

	err = s.PasswordResetRepository.ResetPasswordWithToken(
		hashResetToken(req.ResetToken),
		string(hashedPassword),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("invalid or expired reset token")
	}
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
