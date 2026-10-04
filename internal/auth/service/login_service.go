package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginService struct {
	UserRepository    UserRepository
	SessionRepository SessionRepository
	JWTSecret         string
}

func NewLoginService(
	userRepository UserRepository,
	sessionRepository SessionRepository,
	jwtSecret string,
) *LoginService {
	return &LoginService{
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		JWTSecret:         jwtSecret,
	}
}

func (h *LoginService) Login(
	ctx context.Context,
	req dto.LoginData,
) (string, string, string, bool, error) {

	usernameOrEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if usernameOrEmail == "" {
		usernameOrEmail = strings.ToLower(strings.TrimSpace(req.Username))
	}

	var user *model.User
	var err error

	if strings.Contains(usernameOrEmail, "@") {
		user, err = h.UserRepository.FindUserByEmail(
			ctx,
			usernameOrEmail,
		)
	} else {
		user, err = h.UserRepository.FindUserByUsername(ctx, usernameOrEmail)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", false, ErrInvalidCredentials
	}
	if err != nil {
		log.Printf("find user failed: %v", err)
		return "", "", "", false, ErrInternal
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)

	if err != nil {
		return "", "", "", false, ErrInvalidCredentials
	}

	accessToken, refreshToken, err := createSessionTokens(ctx, h.SessionRepository, h.JWTSecret, user.ID, user.PasswordVersion)
	if err != nil {
		return "", "", "", false, err
	}

	return user.Username, accessToken, refreshToken, user.SkillLevel == nil, nil
}
