package model

import "time"

type Author struct {
	Username    string
	DisplayName *string
	AvatarURL   *string
}

type Post struct {
	ID           int64
	UserID       int64
	Body         string
	CreatedAt    time.Time
	Author       Author
	CommentCount int
	MyReaction   *string
	Reactions    map[string]int
}

type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Body      string
	CreatedAt time.Time
	Author    Author
}
