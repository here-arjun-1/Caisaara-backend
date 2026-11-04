package leaderboard

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

type LocalLeaderboardEntry struct {
	Rank       int    `json:"rank"`
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Rating     int    `json:"rating"`
	DistanceKm int    `json:"distance_km"`
}
