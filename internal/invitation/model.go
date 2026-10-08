package invitation

type Invite struct {
	Code               string `json:"code"`
	GameID             string `json:"game_id"`
	CreatorID          int64  `json:"creator_id"`
	TimeControlMinutes int    `json:"time_control_minutes"`
	IncrementSeconds   int    `json:"increment_seconds"`
	Color              string `json:"color"`
}
