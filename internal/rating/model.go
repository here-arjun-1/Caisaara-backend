package rating

import "time"

type Game struct {
	WhitePlayerID int64
	BlackPlayerID int64
	Mode          string
	Result        string
}

type PlayerRating struct {
	UserID           int64
	Mode             string
	Rating           int
	RatingDeviation  float64
	RatingVolatility float64
	GamesPlayed      int
	Wins             int
	Losses           int
	Draws            int
	BestRating       *int
	BestRatingAt     *time.Time
}
