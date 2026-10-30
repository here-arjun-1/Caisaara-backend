package invitation

import "errors"

var (
	ErrInvalidTimeControl     = errors.New("invalid time control")
	ErrInvalidColor           = errors.New("invalid color")
	ErrInviteNotFound         = errors.New("invite not found or expired")
	ErrSelfJoin               = errors.New("you cannot join your own game")
	ErrAlreadyJoining         = errors.New("someone is already joining this game")
	ErrInviteUsed             = errors.New("invite already used or expired")
	ErrInternal               = errors.New("internal server error")
	ErrChallengeYourself      = errors.New("you cannot challenge yourself")
	ErrOpponentNotFound       = errors.New("opponent not found")
	ErrNotYourChallenge       = errors.New("this challenge is for another player")
	ErrGameNotFound           = errors.New("game not found")
	ErrNotYourGame            = errors.New("you are not playing this game")
	ErrGameNotActive          = errors.New("game is not active")
	ErrWatchLinkNotFound      = errors.New("watch link not found or expired")
	ErrLevelNotPicked         = errors.New("pick your level before playing rated games")
	ErrOpponentLevelNotPicked = errors.New("opponent hasn't picked a level yet, send an unrated challenge")
	ErrPlayerBlocked          = errors.New("you cannot play with this player")
	ErrChallengeAlreadySent   = errors.New("you already have a pending challenge to this player")
	ErrTooManyChallenges      = errors.New("too many pending challenges, please wait")
	ErrChallengeBusy          = errors.New("action in progress, please try again")
)
