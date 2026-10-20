package club

import "errors"

var (
	ErrClubNotFound       = errors.New("club not found")
	ErrClubNameTaken      = errors.New("club name is already taken")
	ErrInvalidClubName    = errors.New("club name must be between 3 and 50 characters")
	ErrDescriptionTooLong = errors.New("description must be at most 500 characters")
	ErrAlreadyMember      = errors.New("you are already a member of this club")
	ErrNotMember          = errors.New("you are not a member of this club")
	ErrEmptyMessage       = errors.New("message cannot be empty")
	ErrMessageTooLong     = errors.New("message must be at most 500 characters")
	ErrRateLimited        = errors.New("rate limit exceeded, please wait")
	ErrOwnerCannotLeave   = errors.New("give leadership to an admin before leaving the club")
	ErrNotAllowed         = errors.New("you are not allowed to do this")
	ErrTargetNotAdmin     = errors.New("leadership can only be given to an admin")
	ErrNotBlocked         = errors.New("player is not blocked")
)
