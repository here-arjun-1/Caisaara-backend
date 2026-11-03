package admin

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Rating    int       `json:"rating"`
	IsBanned  bool      `json:"is_banned"`
	BanReason string    `json:"ban_reason"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	DefaultUsersLimit  = 20
	MaxUsersLimit      = 50
	MaxBanReasonLength = 500
	MaxBanDays         = 365
)
