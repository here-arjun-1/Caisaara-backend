package matchmaking

import "errors"

var (
	ErrInvalidTimeControl = errors.New("invalid time control")
	ErrAlreadyInQueue     = errors.New("you are already in the queue")
	ErrNotInQueue         = errors.New("you are not in the queue")
	ErrGuestRatedGame     = errors.New("guests cannot play rated games")
	ErrTilted             = errors.New("you seem tilted, take a break before your next rated game")
	ErrInternal           = errors.New("internal server error")
)
