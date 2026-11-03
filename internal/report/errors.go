package report

import "errors"

var (
	ErrCannotReportYourself = errors.New("you cannot report yourself")
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidReason        = errors.New("reason must be cheating, abuse or other")
	ErrDetailsTooLong       = errors.New("details must be 500 characters or less")
	ErrReportExists         = errors.New("you already have a pending report against this player")
)
