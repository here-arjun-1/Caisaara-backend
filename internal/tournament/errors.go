package tournament

import "errors"

var (
	ErrTournamentNotFound  = errors.New("tournament not found")
	ErrInvalidName         = errors.New("name is required")
	ErrInvalidFormat       = errors.New("only swiss format is supported")
	ErrInvalidVisibility   = errors.New("visibility must be public or private")
	ErrInvalidMaxPlayers   = errors.New("max_players must be at least 2")
	ErrInvalidTotalRounds  = errors.New("total_rounds must be at least 1")
	ErrInvalidTimeControl  = errors.New("time_control is required")
	ErrUserNotAuthenticated = errors.New("user not authenticated")
)
