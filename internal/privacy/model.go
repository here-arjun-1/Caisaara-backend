package privacy

import (
	"errors"
	"strings"
)

const (
	Everyone    = "everyone"
	Nobody      = "nobody"
	OnlyFriends = "only_friends"
)

var (
	ErrInvalidOption = errors.New("invalid privacy option")
	ErrNotAllowed    = errors.New("not allowed by privacy settings")
)

type Settings struct {
	DirectMessage string `json:"direct_message"`
	Club          string `json:"club"`
	Tournament    string `json:"tournament"`
	InviteFriend  string `json:"invite_friend"`
}

func NormalizeOption(val string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(val))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	switch s {
	case Everyone:
		return Everyone, true
	case Nobody:
		return Nobody, true
	case OnlyFriends, "friends", "friends_only":
		return OnlyFriends, true
	default:
		return "", false
	}
}

func DefaultSettings() Settings {
	return Settings{
		DirectMessage: Everyone,
		Club:          Everyone,
		Tournament:    Everyone,
		InviteFriend:  Everyone,
	}
}
