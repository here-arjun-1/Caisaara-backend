package service

import (
	"context"
	"log/slog"
)

var startingRatings = map[string]int{
	"new":          400,
	"beginner":     800,
	"intermediate": 1200,
	"advanced":     1600,
}

type RatingService struct {
	UserRepository UserRepository
}

func NewRatingService(userRepository UserRepository) *RatingService {
	return &RatingService{
		UserRepository: userRepository,
	}
}

func (s *RatingService) SetInitialRating(ctx context.Context, userID int64, level string) (int, error) {
	rating, ok := startingRatings[level]
	if !ok {
		return 0, ErrInvalidLevel
	}

	updated, err := s.UserRepository.SetInitialRating(ctx, userID, level, rating)
	if err != nil {
		slog.Error("set initial rating failed", "error", err)
		return 0, ErrInternal
	}
	if !updated {
		return 0, ErrRatingAlreadySet
	}

	return rating, nil
}
