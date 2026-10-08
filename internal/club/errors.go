package club

import "errors"

var (
	ErrClubNotFound       = errors.New("club not found")
	ErrClubNameTaken      = errors.New("club name is already taken")
	ErrInvalidClubName    = errors.New("club name must be between 3 and 50 characters")
	ErrDescriptionTooLong = errors.New("description must be at most 500 characters")
	ErrAlreadyMember      = errors.New("you are already a member of this club")
	ErrNotMember          = errors.New("you are not a member of this club")
	ErrOwnerCannotLeave   = errors.New("club owner cannot leave the club")
	ErrEmptyMessage       = errors.New("message cannot be empty")
	ErrMessageTooLong     = errors.New("message must be at most 500 characters")
	ErrRateLimited        = errors.New("rate limit exceeded, please wait")
)
