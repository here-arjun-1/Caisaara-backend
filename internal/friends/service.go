package friends

import (
	"context"
	"log/slog"
	"strings"

	"github.com/here-arjun-1/Caisaara-backend/internal/notification"
)

type Notifier interface {
	Send(ctx context.Context, userID int64, notificationType string, title string, body string, data map[string]any) error
}

type Service struct {
	Repository *Repository
	Notifier   Notifier
	IsOnline   func(userID int64) bool
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SetNotifier(notifier Notifier) {
	s.Notifier = notifier
}

func (s *Service) SetOnlineChecker(isOnline func(userID int64) bool) {
	s.IsOnline = isOnline
}

func (s *Service) SendRequest(ctx context.Context, senderID int64, receiverUsername string) (int64, error) {
	receiverUsername = strings.ToLower(strings.TrimSpace(receiverUsername))
	receiverID, err := s.Repository.FindUserIDByUsername(ctx, receiverUsername)
	if err != nil {
		return 0, err
	}

	if senderID == receiverID {
		return 0, ErrCannotAddYourself
	}

	blocked, err := s.Repository.IsBlockedEitherWay(ctx, senderID, receiverID)
	if err != nil {
		return 0, err
	}
	if blocked {
		return 0, ErrUserBlocked
	}

	requestID, err := s.Repository.CreateRequest(ctx, senderID, receiverID)
	if err != nil {
		return 0, err
	}

	s.notifyFriendRequest(ctx, senderID, receiverID, requestID)
	return requestID, nil
}

func (s *Service) notifyFriendRequest(ctx context.Context, senderID int64, receiverID int64, requestID int64) {
	if s.Notifier == nil {
		return
	}

	senderUsername, err := s.Repository.FindUsernameByID(ctx, senderID)
	if err != nil {
		return
	}

	err = s.Notifier.Send(ctx, receiverID, notification.TypeFriendRequest,
		"New friend request",
		senderUsername+" sent you a friend request",
		map[string]any{"request_id": requestID, "user_id": senderID, "username": senderUsername},
	)
	if err != nil {
		slog.ErrorContext(ctx, "send friend request notification failed", "receiver_id", receiverID, "error", err)
	}
}

func (s *Service) GetPendingRequests(ctx context.Context, userID int64) ([]FriendRequest, error) {
	return s.Repository.FindPendingRequests(ctx, userID)
}

func (s *Service) GetSentRequests(ctx context.Context, userID int64) ([]SentRequest, error) {
	return s.Repository.FindSentRequests(ctx, userID)
}

func (s *Service) AcceptRequest(ctx context.Context, userID int64, requestID int64) error {
	found, err := s.Repository.UpdateRequestStatus(ctx, requestID, userID, StatusAccepted)
	if err != nil {
		return err
	}
	if !found {
		return ErrRequestNotFound
	}

	s.notifyRequestAccepted(ctx, userID, requestID)
	return nil
}

func (s *Service) notifyRequestAccepted(ctx context.Context, userID int64, requestID int64) {
	if s.Notifier == nil {
		return
	}

	senderID, err := s.Repository.FindRequestSenderID(ctx, requestID)
	if err != nil {
		return
	}

	username, err := s.Repository.FindUsernameByID(ctx, userID)
	if err != nil {
		return
	}

	err = s.Notifier.Send(ctx, senderID, notification.TypeFriendAccepted,
		"Friend request accepted",
		username+" accepted your friend request",
		map[string]any{"user_id": userID, "username": username},
	)
	if err != nil {
		slog.ErrorContext(ctx, "send friend accepted notification failed", "sender_id", senderID, "error", err)
	}
}

func (s *Service) RejectRequest(ctx context.Context, userID int64, requestID int64) error {
	found, err := s.Repository.UpdateRequestStatus(ctx, requestID, userID, StatusRejected)
	if err != nil {
		return err
	}
	if !found {
		return ErrRequestNotFound
	}
	return nil
}

func (s *Service) GetFriends(ctx context.Context, userID int64) ([]Friend, error) {
	if s == nil || s.Repository == nil || userID <= 0 {
		return []Friend{}, nil
	}
	friends, err := s.Repository.FindFriends(ctx, userID)
	if err != nil {
		return nil, err
	}
	if friends == nil {
		friends = []Friend{}
	}
	if s.IsOnline != nil {
		for i := range friends {
			if friends[i].UserID > 0 {
				friends[i].IsOnline = s.IsOnline(friends[i].UserID)
			}
		}
	}
	return friends, nil
}

func (s *Service) CancelRequest(ctx context.Context, userID int64, requestID int64) error {
	found, err := s.Repository.DeleteSentRequest(ctx, requestID, userID)
	if err != nil {
		return err
	}
	if !found {
		return ErrRequestNotFound
	}
	return nil
}

func (s *Service) RemoveFriend(ctx context.Context, userID int64, friendID int64) error {
	found, err := s.Repository.DeleteFriend(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFriends
	}
	return nil
}

func (s *Service) GetFriendshipStatus(ctx context.Context, userID int64, username string) (*FriendshipStatus, error) {
	username = strings.ToLower(strings.TrimSpace(username))

	otherID, err := s.Repository.FindUserIDByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if userID == otherID {
		return &FriendshipStatus{Status: FriendshipSelf}, nil
	}

	iBlocked, err := s.Repository.IsBlocked(ctx, userID, otherID)
	if err != nil {
		return nil, err
	}
	if iBlocked {
		return &FriendshipStatus{Status: FriendshipBlocked}, nil
	}

	theyBlocked, err := s.Repository.IsBlocked(ctx, otherID, userID)
	if err != nil {
		return nil, err
	}
	if theyBlocked {
		return &FriendshipStatus{Status: FriendshipUnavailable}, nil
	}

	req, err := s.Repository.FindPairRequest(ctx, userID, otherID)
	if err != nil {
		return nil, err
	}
	if req == nil || req.Status == StatusRejected {
		return &FriendshipStatus{Status: FriendshipNone}, nil
	}
	if req.Status == StatusAccepted {
		return &FriendshipStatus{Status: FriendshipFriends}, nil
	}
	if req.SenderID == userID {
		return &FriendshipStatus{Status: FriendshipRequestSent, RequestID: &req.ID}, nil
	}
	return &FriendshipStatus{Status: FriendshipRequestReceived, RequestID: &req.ID}, nil
}

func (s *Service) BlockUser(ctx context.Context, userID int64, targetID int64) error {
	if userID == targetID {
		return ErrCannotBlockYourself
	}
	return s.Repository.BlockUser(ctx, userID, targetID)
}

func (s *Service) UnblockUser(ctx context.Context, userID int64, targetID int64) error {
	found, err := s.Repository.UnblockUser(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotBlocked
	}
	return nil
}

func (s *Service) GetBlockedUsers(ctx context.Context, userID int64) ([]BlockedUser, error) {
	return s.Repository.FindBlockedUsers(ctx, userID)
}
