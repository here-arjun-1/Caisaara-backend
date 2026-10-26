package notification

import "time"

const (
	TypeFriendRequest  = "friend_request"
	TypeFriendAccepted = "friend_accepted"
	TypeGameInvite     = "game_invite"
)

type Notification struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"-"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Data      map[string]any `json:"data"`
	IsRead    bool           `json:"is_read"`
	CreatedAt time.Time      `json:"created_at"`
}
