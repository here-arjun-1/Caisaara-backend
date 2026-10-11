package friends

import "errors"

var (
	ErrRequestExists     = errors.New("you are already friends or a friend request is already pending")
	ErrCannotAddYourself = errors.New("you cannot send a friend request to yourself")
	ErrUserNotFound      = errors.New("user not found")
	ErrRequestNotFound   = errors.New("friend request not found")
	ErrNotFriends        = errors.New("this user is not in your friends list")
)
