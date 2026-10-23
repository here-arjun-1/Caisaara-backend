package service

import "errors"

var (
	ErrInternal           = errors.New("internal server error")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidDisplayName = errors.New("display_name must be at most 30 characters on a single line")
	ErrInvalidCountry     = errors.New("country must be a 2-letter code like IN")
	ErrInvalidBio         = errors.New("bio must be at most 160 characters")
	ErrInvalidMode        = errors.New("mode must be one of bullet, blitz, rapid, daily")
	ErrInvalidRange       = errors.New("range must be one of 7d, 30d, 90d, 1y, all")
	ErrInvalidPhoto       = errors.New("photo must be a jpeg, png or webp image")
	ErrPhotoTooLarge      = errors.New("photo can be at most 5000x5000 pixels")
	ErrInvalidAvatarID    = errors.New("avatar_id is not a valid default avatar")
	ErrInvalidSearchQuery = errors.New("search query must be 2 to 50 characters")
)
