package bot

import "errors"

var (
	ErrInvalidLevel  = errors.New("invalid bot level")
	ErrInvalidRating = errors.New("bot rating must be between 1000 and 3000")
	ErrNoBestMove    = errors.New("engine returned no move")
)
