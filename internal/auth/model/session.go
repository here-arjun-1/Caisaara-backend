package model

import "time"

type Session struct {
	ID               int64
	UserID           int64
	RefreshTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	RevokedAt        *time.Time
}
