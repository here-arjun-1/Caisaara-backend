package admin

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrCannotBanYourself = errors.New("you cannot ban yourself")
	ErrCannotBanAdmin    = errors.New("you cannot ban another admin")
	ErrReasonRequired    = errors.New("ban reason is required")
	ErrReasonTooLong     = errors.New("ban reason must be 500 characters or less")
	ErrInvalidBanDays    = errors.New("ban days must be between 0 and 365")
)
