package friends

import "time"

const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
)

type FriendRequest struct {
	ID             int64     `json:"id"`
	SenderID       int64     `json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	SenderRating   int       `json:"sender_rating"`
	CreatedAt      time.Time `json:"created_at"`
}

type Friend struct {
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Rating       int       `json:"rating"`
	FriendsSince time.Time `json:"friends_since"`
}
