package admin

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrCannotBanYourself   = errors.New("you cannot ban yourself")
	ErrCannotBanAdmin      = errors.New("you cannot ban another admin")
	ErrReasonRequired      = errors.New("ban reason is required")
	ErrReasonTooLong       = errors.New("ban reason must be 500 characters or less")
	ErrInvalidBanDays      = errors.New("ban days must be between 0 and 365")
	ErrUserNotBanned       = errors.New("user is not banned")
	ErrReportNotFound      = errors.New("report not found")
	ErrInvalidReportStatus = errors.New("status must be pending, resolved or dismissed")
	ErrInvalidResolution   = errors.New("status must be resolved or dismissed")
	ErrReportAlreadyClosed = errors.New("report is already closed")
	ErrNoteTooLong         = errors.New("note must be 500 characters or less")
	ErrPostNotFound        = errors.New("post not found")
	ErrCommentNotFound     = errors.New("comment not found")
	ErrClubNotFound        = errors.New("club not found")
	ErrInvalidClubID       = errors.New("invalid club id")
)
