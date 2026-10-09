package model

import "time"

type Profile struct {
	UserID          int64
	Username        string
	Email           string
	Rating          *int
	RatingDeviation float64
	SkillLevel      *string
	CreatedAt       time.Time
	DisplayName     *string
	Country         *string
	Bio             *string
	AvatarURL       *string
	Ratings         []ModeRating
}

type ModeRating struct {
	Mode            string
	Rating          int
	RatingDeviation float64
	GamesPlayed     int
	BestRating      *int
	BestRatingAt    *time.Time
}

type ModeGameStats struct {
	Wins   int
	Losses int
	Draws  int
}

type RecentGame struct {
	GameID           string
	OpponentUsername string
	Color            string
	Result           string
	EndReason        *string
	EndedAt          *time.Time
}
