package service

import "errors"

var (
	ErrInternal           = errors.New("internal server error")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidDisplayName = errors.New("display_name must be at most 30 characters on a single line")
	ErrInvalidCountry     = errors.New("country must be a 2-letter code like IN")
	ErrInvalidBio         = errors.New("bio must be at most 160 characters")
	ErrInvalidAvatarURL   = errors.New("avatar_url must be an https URL")
	ErrInvalidMode        = errors.New("mode must be one of bullet, blitz, rapid, daily")
)
