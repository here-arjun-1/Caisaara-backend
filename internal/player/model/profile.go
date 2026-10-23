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
	FriendsCount    int
}

type ModeRating struct {
	Mode            string
	Rating          int
	RatingDeviation float64
	GamesPlayed     int
	Wins            int
	Losses          int
	Draws           int
	BestRating      *int
	BestRatingAt    *time.Time
}

type ModeGameStats struct {
	WhiteWins   int
	WhiteLosses int
	WhiteDraws  int
	BlackWins   int
	BlackLosses int
	BlackDraws  int
}

type RecentGame struct {
	GameID           string
	OpponentUsername string
	Color            string
	Result           string
	EndReason        *string
	EndedAt          *time.Time
}

type RatingPoint struct {
	Date   string
	Rating int
}

type LowestRating struct {
	Rating int
	At     *time.Time
}

type GameResult struct {
	Color  string
	Result string
}
