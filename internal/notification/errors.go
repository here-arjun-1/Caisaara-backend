package notification

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrDeadToken            = errors.New("device token is no longer valid")
)
