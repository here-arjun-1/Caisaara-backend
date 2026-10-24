package chat

import "time"

type ChatMessage struct {
	ID        string    `json:"id"`
	GameID    string    `json:"game_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type SendChatMessageRequest struct {
	Message    string `json:"message"`
	ReceiverID int64  `json:"receiver_id,omitempty"`
}

type ChatWSMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id,omitempty"`
	GameID    string    `json:"game_id,omitempty"`
	UserID    int64     `json:"user_id,omitempty"`
	Username  string    `json:"username,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type DirectMessage struct {
	ID         int64     `json:"id"`
	SenderID   int64     `json:"sender_id"`
	ReceiverID int64     `json:"receiver_id"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

type Conversation struct {
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	DisplayName   *string   `json:"display_name"`
	AvatarURL     *string   `json:"avatar_url"`
	LastMessage   string    `json:"last_message"`
	LastMessageAt time.Time `json:"last_message_at"`
	UnreadCount   int       `json:"unread_count"`
}
