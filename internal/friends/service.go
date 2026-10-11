package friends

import "context"

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) SendRequest(ctx context.Context, senderID int64, receiverID int64) (int64, error) {
	if senderID == receiverID {
		return 0, ErrCannotAddYourself
	}

	exists, err := s.Repository.UserExists(ctx, receiverID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, ErrUserNotFound
	}

	return s.Repository.CreateRequest(ctx, senderID, receiverID)
}

func (s *Service) GetPendingRequests(ctx context.Context, userID int64) ([]FriendRequest, error) {
	return s.Repository.FindPendingRequests(ctx, userID)
}

func (s *Service) AcceptRequest(ctx context.Context, userID int64, requestID int64) error {
	found, err := s.Repository.UpdateRequestStatus(ctx, requestID, userID, StatusAccepted)
	if err != nil {
		return err
	}
	if !found {
		return ErrRequestNotFound
	}
	return nil
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
	return s.Repository.FindFriends(ctx, userID)
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
