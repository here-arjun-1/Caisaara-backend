package club

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
)

const (
	defaultClubsLimit = 20
	maxClubsLimit     = 50
	messageHistory    = 50
)

type ClubService interface {
	CreateClub(ctx context.Context, ownerID int64, name string, description string) (*Club, error)
	ListClubs(ctx context.Context, limit int, offset int) ([]Club, error)
	GetClub(ctx context.Context, clubID string) (*Club, error)
	JoinClub(ctx context.Context, clubID string, userID int64) error
	LeaveClub(ctx context.Context, clubID string, userID int64) error
	GetMembers(ctx context.Context, clubID string) ([]Member, error)
	GetLeaderboard(ctx context.Context, clubID string) ([]LeaderboardEntry, error)
	SendMessage(ctx context.Context, clubID string, userID int64, text string) (*Message, error)
	GetMessages(ctx context.Context, clubID string, userID int64) ([]Message, error)
}

type Service struct {
	Repository  ClubRepository
	RateLimiter chat.RateLimiter
}

func NewService(repository ClubRepository, rateLimiter chat.RateLimiter) ClubService {
	return &Service{
		Repository:  repository,
		RateLimiter: rateLimiter,
	}
}

func (s *Service) CreateClub(ctx context.Context, ownerID int64, name string, description string) (*Club, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	nameLength := utf8.RuneCountInString(name)
	if nameLength < MinNameLength || nameLength > MaxNameLength {
		return nil, ErrInvalidClubName
	}

	if utf8.RuneCountInString(description) > MaxDescriptionLength {
		return nil, ErrDescriptionTooLong
	}

	club := &Club{
		Name:        name,
		Description: description,
		OwnerID:     ownerID,
	}

	if err := s.Repository.CreateClub(ctx, club); err != nil {
		return nil, err
	}

	return club, nil
}

func (s *Service) ListClubs(ctx context.Context, limit int, offset int) ([]Club, error) {
	if limit <= 0 {
		limit = defaultClubsLimit
	}
	if limit > maxClubsLimit {
		limit = maxClubsLimit
	}
	if offset < 0 {
		offset = 0
	}

	return s.Repository.ListClubs(ctx, limit, offset)
}

func (s *Service) GetClub(ctx context.Context, clubID string) (*Club, error) {
	return s.Repository.FindClubByID(ctx, clubID)
}

func (s *Service) JoinClub(ctx context.Context, clubID string, userID int64) error {
	if _, err := s.Repository.FindClubByID(ctx, clubID); err != nil {
		return err
	}

	return s.Repository.AddMember(ctx, clubID, userID)
}

func (s *Service) LeaveClub(ctx context.Context, clubID string, userID int64) error {
	if _, err := s.Repository.FindClubByID(ctx, clubID); err != nil {
		return err
	}

	role, err := s.Repository.GetMemberRole(ctx, clubID, userID)
	if err != nil {
		return err
	}

	if role == RoleOwner {
		return ErrOwnerCannotLeave
	}

	return s.Repository.RemoveMember(ctx, clubID, userID)
}

func (s *Service) GetMembers(ctx context.Context, clubID string) ([]Member, error) {
	if _, err := s.Repository.FindClubByID(ctx, clubID); err != nil {
		return nil, err
	}

	return s.Repository.ListMembers(ctx, clubID)
}

func (s *Service) GetLeaderboard(ctx context.Context, clubID string) ([]LeaderboardEntry, error) {
	if _, err := s.Repository.FindClubByID(ctx, clubID); err != nil {
		return nil, err
	}

	return s.Repository.GetLeaderboard(ctx, clubID)
}

func (s *Service) SendMessage(ctx context.Context, clubID string, userID int64, text string) (*Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyMessage
	}
	if utf8.RuneCountInString(text) > MaxMessageLength {
		return nil, ErrMessageTooLong
	}

	if err := s.checkMember(ctx, clubID, userID); err != nil {
		return nil, err
	}

	if s.RateLimiter != nil {
		allowed, _ := s.RateLimiter.Allow(ctx, clubID, userID)
		if !allowed {
			return nil, ErrRateLimited
		}
	}

	msg := &Message{
		ClubID:  clubID,
		UserID:  userID,
		Message: text,
	}

	if err := s.Repository.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *Service) GetMessages(ctx context.Context, clubID string, userID int64) ([]Message, error) {
	if err := s.checkMember(ctx, clubID, userID); err != nil {
		return nil, err
	}

	return s.Repository.GetRecentMessages(ctx, clubID, messageHistory)
}

func (s *Service) checkMember(ctx context.Context, clubID string, userID int64) error {
	if _, err := s.Repository.FindClubByID(ctx, clubID); err != nil {
		return err
	}

	_, err := s.Repository.GetMemberRole(ctx, clubID, userID)
	return err
}
