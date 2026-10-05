package dto

import "time"

type CreatePostData struct {
	Body string `json:"body" binding:"required"`
}

type CreateCommentData struct {
	Body string `json:"body" binding:"required"`
}

type SetReactionData struct {
	Reaction string `json:"reaction" binding:"required,oneof=like love fire wow"`
}

type AuthorResponse struct {
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type PostResponse struct {
	ID           int64          `json:"id"`
	Author       AuthorResponse `json:"author"`
	Body         string         `json:"body"`
	Reactions    map[string]int `json:"reactions"`
	MyReaction   *string        `json:"my_reaction"`
	CommentCount int            `json:"comment_count"`
	CreatedAt    time.Time      `json:"created_at"`
}

type FeedResponse struct {
	Posts        []PostResponse `json:"posts"`
	NextBeforeID *int64         `json:"next_before_id"`
}

type CommentResponse struct {
	ID        int64          `json:"id"`
	Author    AuthorResponse `json:"author"`
	Body      string         `json:"body"`
	CreatedAt time.Time      `json:"created_at"`
}

type CommentListResponse struct {
	Comments    []CommentResponse `json:"comments"`
	NextAfterID *int64            `json:"next_after_id"`
}

type ReactionResponse struct {
	Reactions  map[string]int `json:"reactions"`
	MyReaction *string        `json:"my_reaction"`
}
