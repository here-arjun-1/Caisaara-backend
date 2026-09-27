package model

import "time"

type PasswordReset struct {
	ID                  int64
	Email               string
	OTPHash             string
	OTPExpiresAt        time.Time
	ResetTokenHash      *string
	ResetTokenExpiresAt *time.Time
	CreatedAt           time.Time
}
