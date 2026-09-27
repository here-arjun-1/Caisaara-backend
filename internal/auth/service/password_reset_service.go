package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

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
	user, err := s.UserRepository.FindUserByEmail(req.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // Don't leak user existence
	}
	if err != nil {
		log.Printf("find user by email failed: %v", err)
		return ErrInternal
	}

	otp, err := generateOTP()
	if err != nil {
		log.Printf("generate otp failed: %v", err)
		return ErrInternal
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash otp failed: %v", err)
		return ErrInternal
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	err = s.PasswordResetRepository.SaveOTP(user.Email, string(otpHash), expiresAt)
	if err != nil {
		log.Printf("save otp failed: %v", err)
		return ErrInternal
	}

	err = email.SendPasswordResetEmail(user.Email, otp)
	if err != nil {
		log.Printf("send password reset email failed: %v", err)
		return ErrInternal
	}

	return nil
}

func (s *PasswordResetService) VerifyCode(req dto.VerifyCodeRequest) (string, error) {
	pr, err := s.PasswordResetRepository.FindByEmail(req.Email)
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
	tokenHash := hashResetToken(req.ResetToken)
	pr, err := s.PasswordResetRepository.FindByResetTokenHash(tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("invalid or expired reset token")
	}
	if err != nil {
		log.Printf("find password reset by token failed: %v", err)
		return ErrInternal
	}

	if pr.ResetTokenExpiresAt == nil || time.Now().After(*pr.ResetTokenExpiresAt) {
		return errors.New("reset token expired")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash new password failed: %v", err)
		return ErrInternal
	}

	err = s.UserRepository.UpdatePassword(pr.Email, string(hashedPassword))
	if err != nil {
		log.Printf("update password failed: %v", err)
		return ErrInternal
	}

	err = s.PasswordResetRepository.DeletePasswordReset(pr.Email)
	if err != nil {
		log.Printf("delete password reset failed: %v", err)
	}

	user, err := s.UserRepository.FindUserByEmail(pr.Email)
	if err == nil {
		_ = s.SessionRepository.RevokeAllSessions(user.ID)
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
