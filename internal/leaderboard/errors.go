package leaderboard

import "errors"

var (
	ErrInvalidMode      = errors.New("mode must be bullet, blitz, rapid or daily")
	ErrInvalidRadius    = errors.New("radius_km must be between 1 and 100")
	ErrLocationNotSaved = errors.New("save your location first")
	ErrNearbyOff        = errors.New("turn on nearby to see the local leaderboard")
	ErrLocationTooOld   = errors.New("your location is too old, update it to see the local leaderboard")
)
