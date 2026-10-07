package bot

import "errors"

var (
	ErrInvalidLevel       = errors.New("invalid bot level")
	ErrInvalidRating      = errors.New("bot rating must be between 1000 and 3000")
	ErrNoBestMove         = errors.New("engine returned no move")
	ErrGameNotFound       = errors.New("bot game not found")
	ErrTooManyActiveGames = errors.New("you can play at most 5 bot games at a time")
	ErrGameChanged        = errors.New("game was updated by another request")
)
