package nearby

import "errors"

var (
	ErrInvalidLocation  = errors.New("latitude must be between -90 and 90 and longitude between -180 and 180")
	ErrLocationNotSaved = errors.New("save your location first")
	ErrLocationTooOld   = errors.New("your location is too old, update it to see players around you")
	ErrNearbyOff        = errors.New("turn on nearby to see players around you")
)
