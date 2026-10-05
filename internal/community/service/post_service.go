package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/here-arjun-1/Caisaara-backend/internal/community/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/community/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultPageSize             = 20
	maxPageSize                 = 50
	postgresForeignKeyViolation = "23503"
)

type PostRepository interface {
	CreatePost(ctx context.Context, userID int64, body string) (int64, error)
	FindPostByID(ctx context.Context, viewerID, postID int64) (*model.Post, error)
	ListPosts(ctx context.Context, viewerID, beforeID int64, limit int) ([]*model.Post, error)
	FindPostOwner(ctx context.Context, postID int64) (int64, error)
	DeletePost(ctx context.Context, postID int64) error
	CountReactions(ctx context.Context, postIDs []int64) (map[int64]map[string]int, error)
	SetReaction(ctx context.Context, postID, userID int64, reaction string) error
	DeleteReaction(ctx context.Context, postID, userID int64) error
}

type PostService struct {
	PostRepository PostRepository
}

func NewPostService(postRepository PostRepository) *PostService {
	return &PostService{
		PostRepository: postRepository,
	}
}

func (s *PostService) CreatePost(ctx context.Context, userID int64, req dto.CreatePostData) (*dto.PostResponse, error) {
	body := strings.TrimSpace(req.Body)
	length := utf8.RuneCountInString(body)
	if length < 1 || length > 2000 {
		return nil, ErrInvalidBody
	}

	postID, err := s.PostRepository.CreatePost(ctx, userID, body)
	if err != nil {
		slog.Error("create post failed", "error", err)
		return nil, ErrInternal
	}

	return s.GetPost(ctx, userID, postID)
}

func (s *PostService) GetPost(ctx context.Context, viewerID, postID int64) (*dto.PostResponse, error) {
	p, err := s.PostRepository.FindPostByID(ctx, viewerID, postID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		slog.Error("find post failed", "error", err)
		return nil, ErrInternal
	}

	counts, err := s.PostRepository.CountReactions(ctx, []int64{p.ID})
	if err != nil {
		slog.Error("count reactions failed", "error", err)
		return nil, ErrInternal
	}
	p.Reactions = counts[p.ID]

	response := toPostResponse(p)
	return &response, nil
}

func (s *PostService) GetFeed(ctx context.Context, viewerID, beforeID int64, limit int) (*dto.FeedResponse, error) {
	limit = clampLimit(limit)

	posts, err := s.PostRepository.ListPosts(ctx, viewerID, beforeID, limit)
	if err != nil {
		slog.Error("list posts failed", "error", err)
		return nil, ErrInternal
	}

	postIDs := make([]int64, 0, len(posts))
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}

	counts, err := s.PostRepository.CountReactions(ctx, postIDs)
	if err != nil {
		slog.Error("count reactions failed", "error", err)
		return nil, ErrInternal
	}

	feed := &dto.FeedResponse{
		Posts: make([]dto.PostResponse, 0, len(posts)),
	}
	for _, p := range posts {
		p.Reactions = counts[p.ID]
		feed.Posts = append(feed.Posts, toPostResponse(p))
	}

	if len(posts) == limit {
		lastID := posts[len(posts)-1].ID
		feed.NextBeforeID = &lastID
	}

	return feed, nil
}

func (s *PostService) DeletePost(ctx context.Context, userID, postID int64) error {
	ownerID, err := s.PostRepository.FindPostOwner(ctx, postID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPostNotFound
	}
	if err != nil {
		slog.Error("find post owner failed", "error", err)
		return ErrInternal
	}

	if ownerID != userID {
		return ErrNotAllowed
	}

	if err := s.PostRepository.DeletePost(ctx, postID); err != nil {
		slog.Error("delete post failed", "error", err)
		return ErrInternal
	}

	return nil
}

func (s *PostService) SetReaction(ctx context.Context, userID, postID int64, reaction string) (*dto.ReactionResponse, error) {
	err := s.PostRepository.SetReaction(ctx, postID, userID, reaction)
	if isForeignKeyError(err) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		slog.Error("set reaction failed", "error", err)
		return nil, ErrInternal
	}

	return s.reactionSummary(ctx, postID, &reaction)
}

func (s *PostService) DeleteReaction(ctx context.Context, userID, postID int64) (*dto.ReactionResponse, error) {
	if _, err := s.PostRepository.FindPostOwner(ctx, postID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		slog.Error("find post owner failed", "error", err)
		return nil, ErrInternal
	}

	if err := s.PostRepository.DeleteReaction(ctx, postID, userID); err != nil {
		slog.Error("delete reaction failed", "error", err)
		return nil, ErrInternal
	}

	return s.reactionSummary(ctx, postID, nil)
}

func (s *PostService) reactionSummary(ctx context.Context, postID int64, myReaction *string) (*dto.ReactionResponse, error) {
	counts, err := s.PostRepository.CountReactions(ctx, []int64{postID})
	if err != nil {
		slog.Error("count reactions failed", "error", err)
		return nil, ErrInternal
	}

	return &dto.ReactionResponse{
		Reactions:  nonNilCounts(counts[postID]),
		MyReaction: myReaction,
	}, nil
}

func toPostResponse(p *model.Post) dto.PostResponse {
	return dto.PostResponse{
		ID: p.ID,
		Author: dto.AuthorResponse{
			Username:    p.Author.Username,
			DisplayName: p.Author.DisplayName,
			AvatarURL:   p.Author.AvatarURL,
		},
		Body:         p.Body,
		Reactions:    nonNilCounts(p.Reactions),
		MyReaction:   p.MyReaction,
		CommentCount: p.CommentCount,
		CreatedAt:    p.CreatedAt,
	}
}

func nonNilCounts(counts map[string]int) map[string]int {
	if counts == nil {
		return map[string]int{}
	}
	return counts
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultPageSize
	}
	if limit > maxPageSize {
		return maxPageSize
	}
	return limit
}

func isForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == postgresForeignKeyViolation
}
