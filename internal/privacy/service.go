package privacy

import (
	"context"
)

type PrivacyRepository interface {
	GetSettings(ctx context.Context, userID int64) (*Settings, error)
	UpsertSettings(ctx context.Context, userID int64, s Settings) (*Settings, error)
	AreFriends(ctx context.Context, user1ID, user2ID int64) (bool, error)
}

type Service struct {
	repo PrivacyRepository
}

func NewService(repo PrivacyRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSettings(ctx context.Context, userID int64) (*Settings, error) {
	return s.repo.GetSettings(ctx, userID)
}

func (s *Service) UpdateSettings(ctx context.Context, userID int64, req Settings) (*Settings, error) {
	current, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.DirectMessage != "" {
		norm, ok := NormalizeOption(req.DirectMessage)
		if !ok {
			return nil, ErrInvalidOption
		}
		current.DirectMessage = norm
	}

	if req.Club != "" {
		norm, ok := NormalizeOption(req.Club)
		if !ok {
			return nil, ErrInvalidOption
		}
		current.Club = norm
	}

	if req.Tournament != "" {
		norm, ok := NormalizeOption(req.Tournament)
		if !ok {
			return nil, ErrInvalidOption
		}
		current.Tournament = norm
	}

	if req.InviteFriend != "" {
		norm, ok := NormalizeOption(req.InviteFriend)
		if !ok {
			return nil, ErrInvalidOption
		}
		current.InviteFriend = norm
	}

	return s.repo.UpsertSettings(ctx, userID, *current)
}

func (s *Service) check(ctx context.Context, option string, actorID, targetID int64) error {
	if actorID == targetID || option == Everyone || option == "" {
		return nil
	}
	if option == Nobody {
		return ErrNotAllowed
	}
	if option == OnlyFriends {
		isFriend, err := s.repo.AreFriends(ctx, actorID, targetID)
		if err != nil {
			return err
		}
		if !isFriend {
			return ErrNotAllowed
		}
	}
	return nil
}

func (s *Service) CheckDirectMessage(ctx context.Context, senderID, receiverID int64) error {
	settings, err := s.repo.GetSettings(ctx, receiverID)
	if err != nil {
		return err
	}
	return s.check(ctx, settings.DirectMessage, senderID, receiverID)
}

func (s *Service) CheckInviteFriend(ctx context.Context, senderID, opponentID int64) error {
	settings, err := s.repo.GetSettings(ctx, opponentID)
	if err != nil {
		return err
	}
	return s.check(ctx, settings.InviteFriend, senderID, opponentID)
}

func (s *Service) CheckClubJoin(ctx context.Context, joinerID, ownerID int64) error {
	settings, err := s.repo.GetSettings(ctx, ownerID)
	if err != nil {
		return err
	}
	return s.check(ctx, settings.Club, joinerID, ownerID)
}

func (s *Service) CheckTournamentJoin(ctx context.Context, joinerID, creatorID int64) error {
	settings, err := s.repo.GetSettings(ctx, creatorID)
	if err != nil {
		return err
	}
	return s.check(ctx, settings.Tournament, joinerID, creatorID)
}
