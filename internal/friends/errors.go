package friends

import "errors"

var (
	ErrRequestExists       = errors.New("you are already friends or a friend request is already pending")
	ErrCannotAddYourself   = errors.New("you cannot send a friend request to yourself")
	ErrUserNotFound        = errors.New("user not found")
	ErrRequestNotFound     = errors.New("friend request not found")
	ErrNotFriends          = errors.New("this user is not in your friends list")
	ErrCannotBlockYourself = errors.New("you cannot block yourself")
	ErrNotBlocked          = errors.New("this user is not in your blocked list")
	ErrUserBlocked         = errors.New("you cannot send a friend request to this user")
)
