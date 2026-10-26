package leaderboard

import "errors"

var (
	ErrInvalidMode = errors.New("mode must be bullet, blitz, rapid or daily")
)
