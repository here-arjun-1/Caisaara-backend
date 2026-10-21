package club

import "time"

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

const (
	MinNameLength        = 3
	MaxNameLength        = 50
	MaxDescriptionLength = 500
	MaxMessageLength     = 500
)

type Club struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     int64     `json:"owner_id"`
	ShareCode   string    `json:"share_code"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type MyClub struct {
	Club
	Role string `json:"role"`
}

type Member struct {
	UserID   int64     `json:"user_id"`
	Username string    `json:"username"`
	Rating   int       `json:"rating"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

type Message struct {
	ID        string    `json:"id"`
	ClubID    string    `json:"club_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type BlockedUser struct {
	UserID       int64      `json:"user_id"`
	Username     string     `json:"username"`
	BlockedBy    *int64     `json:"blocked_by"`
	BlockedUntil *time.Time `json:"blocked_until"`
	CreatedAt    time.Time  `json:"created_at"`
}
