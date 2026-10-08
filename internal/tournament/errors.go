package tournament

import "errors"

var (
	ErrTournamentNotFound   = errors.New("tournament not found")
	ErrInvalidName          = errors.New("name is required")
	ErrInvalidFormat        = errors.New("format must be swiss, round_robin, or knockout")
	ErrInvalidVisibility    = errors.New("visibility must be public or private")
	ErrInvalidMinPlayers    = errors.New("min_players must be at least 2 and cannot exceed max_players")
	ErrInvalidMaxPlayers    = errors.New("max_players must be at least 2")
	ErrInvalidTotalRounds   = errors.New("total_rounds must be at least 1")
	ErrInvalidTimeControl   = errors.New("time_control is required")
	ErrUserNotAuthenticated  = errors.New("user not authenticated")
	ErrNotRegistration      = errors.New("tournament is not accepting registrations")
	ErrTournamentFull       = errors.New("tournament is full")
	ErrAlreadyJoined        = errors.New("user has already joined this tournament")
	ErrInvalidInviteCode    = errors.New("invalid or missing invite code for private tournament")
	ErrNotJoined            = errors.New("user is not joined in this tournament")
	ErrCannotLeaveStarted   = errors.New("cannot leave a tournament that has already started or finished")
	ErrNotCreator           = errors.New("only the tournament creator can start the tournament")
	ErrNotEnoughPlayers     = errors.New("at least 2 players are required to start the tournament")
)
