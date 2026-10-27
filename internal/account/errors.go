package account

import "errors"

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrWrongPassword = errors.New("password is incorrect")
	ErrInternal      = errors.New("internal server error")
)
