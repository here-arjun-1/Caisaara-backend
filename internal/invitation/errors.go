package invitation

import "errors"

var (
	ErrInvalidTimeControl = errors.New("invalid time control")
	ErrInvalidColor       = errors.New("invalid color")
	ErrInviteNotFound     = errors.New("invite not found or expired")
	ErrSelfJoin           = errors.New("you cannot join your own game")
	ErrAlreadyJoining     = errors.New("someone is already joining this game")
	ErrInviteUsed         = errors.New("invite already used or expired")
	ErrInternal           = errors.New("internal server error")
)
