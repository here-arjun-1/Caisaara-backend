package service

import "errors"

var (
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidUsername    = errors.New("username can only contain letters, numbers and underscore")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInternal           = errors.New("internal server error")
)
