package nearby

import (
	"context"
	"errors"
	"math"
)

var (
	ErrInvalidLocation  = errors.New("latitude must be between -90 and 90 and longitude between -180 and 180")
	ErrLocationNotSaved = errors.New("save your location first")
	ErrNearbyOff        = errors.New("turn on nearby to see players around you")
)

type Service struct {
	Repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		Repository: repository,
	}
}

func (s *Service) UpdateLocation(ctx context.Context, userID int64, latitude float64, longitude float64) error {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return ErrInvalidLocation
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
	enabled, err := s.Repository.IsNearbyEnabled(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrNearbyOff
	}

	return s.Repository.FindNearbyPlayers(ctx, userID)
}
