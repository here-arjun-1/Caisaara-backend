package invitation

type Invite struct {
	Code               string `json:"code"`
	GameID             string `json:"game_id"`
	CreatorID          int64  `json:"creator_id"`
	TimeControlMinutes int    `json:"time_control_minutes"`
	IncrementSeconds   int    `json:"increment_seconds"`
	Color              string `json:"color"`
	Status             string `json:"status,omitempty"`
	OpponentID         int64  `json:"opponent_id,omitempty"`
	Rated              bool   `json:"rated"`
}

type Challenge struct {
	Code               string `json:"code"`
	GameID             string `json:"game_id"`
	ChallengerID       int64  `json:"challenger_id"`
	ChallengerUsername string `json:"challenger_username"`
	ChallengerColor    string `json:"challenger_color"`
	TimeControlMinutes int    `json:"time_control_minutes"`
	IncrementSeconds   int    `json:"increment_seconds"`
}
