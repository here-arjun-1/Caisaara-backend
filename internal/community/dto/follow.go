package dto

import "time"

type FollowCountsResponse struct {
	Followers int `json:"followers"`
	Following int `json:"following"`
}

type FollowStatusResponse struct {
	IsFollowing bool `json:"is_following"`
}

type UserListResponse struct {
	Users        []AuthorResponse `json:"users"`
	NextBeforeID *int64           `json:"next_before_id,omitempty"`
}

type LiveGamePlayer struct {
	ID        int64   `json:"id"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatar_url"`
	Rating    int     `json:"rating"`
}

type FollowingLiveGame struct {
	GameID    string         `json:"game_id"`
	WatchCode string         `json:"watch_code"`
	White     LiveGamePlayer `json:"white"`
	Black     LiveGamePlayer `json:"black"`
	Mode      string         `json:"mode"`
	StartedAt *time.Time     `json:"started_at"`
}

type FollowingLiveGamesResponse struct {
	Games []FollowingLiveGame `json:"games"`
}
