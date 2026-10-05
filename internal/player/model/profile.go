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
}
