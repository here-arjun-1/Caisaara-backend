package chat

import "time"

type ChatMessage struct {
	ID        string    `json:"id"`
	GameID    string    `json:"game_id"`
	UserID    int64     `json:"user_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type SendChatMessageRequest struct {
	Message string `json:"message"`
}

type ChatWSMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id,omitempty"`
	GameID    string    `json:"game_id,omitempty"`
	UserID    int64     `json:"user_id,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}
