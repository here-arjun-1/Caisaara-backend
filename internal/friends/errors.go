package friends

import "errors"

var (
	ErrRequestExists = errors.New("you are already friends or a friend request is already pending")
)
