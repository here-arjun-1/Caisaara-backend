package service

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/model"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	FindUserByID(ctx context.Context, id int64) (*model.User, error)
	FindUserByEmail(ctx context.Context, email string) (*model.User, error)
	FindUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUserTx(ctx context.Context, tx pgx.Tx, user *model.User) error
	SetInitialRating(ctx context.Context, userID int64, level string, rating int) (bool, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *model.Session) error
	CreateSessionTx(ctx context.Context, tx pgx.Tx, session *model.Session) error
	FindSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*model.Session, error)
	RotateSession(ctx context.Context, oldRefreshTokenHash string, newSession *model.Session) error
	RevokeSession(ctx context.Context, refreshTokenHash string) error
	RevokeAllSessions(ctx context.Context, userID int64) error
	RevokeAllSessionsByTokenHash(ctx context.Context, refreshTokenHash string) error
}

type PasswordResetRepository interface {
	ResetPassword(ctx context.Context, email, hashedPassword string) error
}
