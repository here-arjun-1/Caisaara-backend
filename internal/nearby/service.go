package nearby

import (
	"context"
	"math"
)

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) UpdateLocation(ctx context.Context, userID int64, latitude float64, longitude float64) (bool, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return false, ErrInvalidLocation
	}

	latitude = math.Round(latitude*100) / 100
	longitude = math.Round(longitude*100) / 100

	return s.Repository.SaveLocation(ctx, userID, latitude, longitude)
}

func (s *Service) SetNearbyEnabled(ctx context.Context, userID int64, enabled bool) error {
	found, err := s.Repository.SetNearbyEnabled(ctx, userID, enabled)
	if err != nil {
		return err
	}
	if !found {
		return ErrLocationNotSaved
	}
	return nil
}

func (s *Service) GetNearbyPlayers(ctx context.Context, userID int64) ([]NearbyPlayer, error) {
	enabled, locationFresh, err := s.Repository.GetNearbyStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrNearbyOff
	}
	if !locationFresh {
		return nil, ErrLocationTooOld
	}

	return s.Repository.FindNearbyPlayers(ctx, userID)
}
