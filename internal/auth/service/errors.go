package service

import "errors"

var (
	ErrUsernameTaken      = errors.New("username already taken")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInternal           = errors.New("internal server error")
	ErrRatingAlreadySet   = errors.New("rating already set")
	ErrInvalidLevel       = errors.New("invalid level")
)
