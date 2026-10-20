package club

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/here-arjun-1/Caisaara-backend/internal/chat"
)

const (
	defaultClubsLimit = 20
	maxClubsLimit     = 50
	messageHistory    = 50
	kickBlockDuration = 7 * 24 * time.Hour
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
	PromoteToAdmin(ctx context.Context, clubID string, leaderID int64, targetID int64) error
	RemoveAdmin(ctx context.Context, clubID string, leaderID int64, targetID int64) error
	KickMember(ctx context.Context, clubID string, userID int64, targetID int64) error
	TransferLeadership(ctx context.Context, clubID string, leaderID int64, targetID int64) error
	BanMember(ctx context.Context, clubID string, userID int64, targetID int64) error
	UnblockMember(ctx context.Context, clubID string, userID int64, targetID int64) error
	GetBlockedList(ctx context.Context, clubID string, userID int64) ([]BlockedUser, error)
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

	blocked, blockedUntil, err := s.Repository.IsBlocked(ctx, clubID, userID)
	if err != nil {
		return err
	}
	if blocked && blockedUntil == nil {
		return ErrBannedFromClub
	}
	if blocked {
		return ErrKickedRecently
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

func (s *Service) getRole(ctx context.Context, clubID string, userID int64) (string, error) {
	_, err := s.Repository.FindClubByID(ctx, clubID)
	if err != nil {
		return "", err
	}

	return s.Repository.GetMemberRole(ctx, clubID, userID)
}

func (s *Service) PromoteToAdmin(ctx context.Context, clubID string, leaderID int64, targetID int64) error {
	myRole, err := s.getRole(ctx, clubID, leaderID)
	if err != nil {
		return err
	}
	if myRole != RoleOwner {
		return ErrNotAllowed
	}

	targetRole, err := s.Repository.GetMemberRole(ctx, clubID, targetID)
	if err != nil {
		return err
	}

	if targetRole != RoleMember {
		return ErrNotAllowed
	}

	return s.Repository.UpdateMemberRole(ctx, clubID, targetID, RoleAdmin)
}

func (s *Service) RemoveAdmin(ctx context.Context, clubID string, leaderID int64, targetID int64) error {
	myRole, err := s.getRole(ctx, clubID, leaderID)
	if err != nil {
		return err
	}
	if myRole != RoleOwner {
		return ErrNotAllowed
	}
	targetRole, err := s.Repository.GetMemberRole(ctx, clubID, targetID)
	if err != nil {
		return err
	}
	if targetRole != RoleAdmin {
		return ErrNotAllowed
	}
	return s.Repository.UpdateMemberRole(ctx, clubID, targetID, RoleMember)
}

func (s *Service) KickMember(ctx context.Context, clubID string, userID int64, targetID int64) error {
	if userID == targetID {
		return ErrNotAllowed
	}
	myRole, err := s.getRole(ctx, clubID, userID)
	if err != nil {
		return err
	}
	targetRole, err := s.Repository.GetMemberRole(ctx, clubID, targetID)
	if err != nil {
		return err
	}
	canKick := myRole == RoleOwner || (myRole == RoleAdmin && targetRole == RoleMember)
	if !canKick {
		return ErrNotAllowed
	}

	blockedUntil := time.Now().Add(kickBlockDuration)
	return s.Repository.BlockUser(ctx, clubID, targetID, userID, &blockedUntil)
}

func (s *Service) TransferLeadership(ctx context.Context, clubID string, leaderID int64, targetID int64) error {
	if leaderID == targetID {
		return ErrNotAllowed
	}
	myRole, err := s.getRole(ctx, clubID, leaderID)
	if err != nil {
		return err
	}
	if myRole != RoleOwner {
		return ErrNotAllowed
	}
	targetRole, err := s.Repository.GetMemberRole(ctx, clubID, targetID)
	if err != nil {
		return err
	}
	if targetRole != RoleAdmin {
		return ErrTargetNotAdmin
	}
	return s.Repository.TransferOwnership(ctx, clubID, leaderID, targetID)
}

func (s *Service) BanMember(ctx context.Context, clubID string, userID int64, targetID int64) error {
	if userID == targetID {
		return ErrNotAllowed
	}
	myRole, err := s.getRole(ctx, clubID, userID)
	if err != nil {
		return err
	}

	targetRole, err := s.Repository.GetMemberRole(ctx, clubID, targetID)
	if err != nil && !errors.Is(err, ErrNotMember) {
		return err
	}

	canBan := myRole == RoleOwner || (myRole == RoleAdmin && (targetRole == RoleMember || targetRole == ""))
	if !canBan {
		return ErrNotAllowed
	}

	return s.Repository.BlockUser(ctx, clubID, targetID, userID, nil)
}

func (s *Service) UnblockMember(ctx context.Context, clubID string, userID int64, targetID int64) error {
	myRole, err := s.getRole(ctx, clubID, userID)
	if err != nil {
		return err
	}
	if myRole != RoleOwner && myRole != RoleAdmin {
		return ErrNotAllowed
	}

	return s.Repository.RemoveBlock(ctx, clubID, targetID)
}

func (s *Service) GetBlockedList(ctx context.Context, clubID string, userID int64) ([]BlockedUser, error) {
	myRole, err := s.getRole(ctx, clubID, userID)
	if err != nil {
		return nil, err
	}
	if myRole != RoleOwner && myRole != RoleAdmin {
		return nil, ErrNotAllowed
	}

	return s.Repository.ListBlocked(ctx, clubID)
}
