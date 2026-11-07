package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/worker"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy_data"), bcrypt.DefaultCost)

type LoginService struct {
	UserRepository    UserRepository
	SessionRepository SessionRepository
	TaskDistributor   *asynq.Client
	JWTSecret         string
}

func NewLoginService(
	userRepository UserRepository,
	sessionRepository SessionRepository,
	taskDistributor *asynq.Client,
	jwtSecret string,
) *LoginService {
	return &LoginService{
		UserRepository:    userRepository,
		SessionRepository: sessionRepository,
		TaskDistributor:   taskDistributor,
		JWTSecret:         jwtSecret,
	}
}

func (h *LoginService) sendAdminLoginAlert(ctx context.Context, user *model.User, ip string) {
	if h.TaskDistributor == nil {
		return
	}

	task, err := worker.NewEmailAdminLoginTask(user.Email, ip, time.Now().UTC())
	if err != nil {
		slog.Error("could not create admin login alert task", "error", err)
		return
	}

	if _, err := h.TaskDistributor.EnqueueContext(ctx, task); err != nil {
		slog.Error("could not enqueue admin login alert task", "error", err)
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
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return "", "", "", false, ErrInvalidCredentials
	}
	if err != nil {
		slog.Error("find user failed", "error", err)
		return "", "", "", false, ErrInternal
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)

	if err != nil {
		return "", "", "", false, ErrInvalidCredentials
	}

	if user.IsBanned {
		return "", "", "", false, ErrUserBanned
	}

	accessToken, refreshToken, err := createSessionTokens(ctx, h.SessionRepository, h.JWTSecret, user.ID, user.PasswordVersion)
	if err != nil {
		return "", "", "", false, err
	}

	if user.Role == "admin" {
		h.sendAdminLoginAlert(ctx, user, req.IP)
	}

	return user.Username, accessToken, refreshToken, user.SkillLevel == nil, nil
}
