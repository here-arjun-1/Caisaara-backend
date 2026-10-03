package service

import (
	"log"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
)

var startingRatings = map[string]int{
	"new":          400,
	"beginner":     800,
	"intermediate": 1200,
	"advanced":     1600,
}

type RatingService struct {
	UserRepository *repository.UserRepository
}

func NewRatingService(userRepository *repository.UserRepository) *RatingService {
	return &RatingService{
		UserRepository: userRepository,
	}
}

func (s *RatingService) SetInitialRating(userID int64, level string) (int, error) {
	rating, ok := startingRatings[level]
	if !ok {
		return 0, ErrInvalidLevel
	}

	updated, err := s.UserRepository.SetInitialRating(userID, level, rating)
	if err != nil {
		log.Printf("set initial rating failed: %v", err)
		return 0, ErrInternal
	}
	if !updated {
		return 0, ErrRatingAlreadySet
	}

	return rating, nil
}
