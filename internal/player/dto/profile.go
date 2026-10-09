package dto

import "time"

type MyProfileResponse struct {
	Username    string                        `json:"username"`
	Email       string                        `json:"email"`
	DisplayName *string                       `json:"display_name"`
	Country     *string                       `json:"country"`
	Bio         *string                       `json:"bio"`
	AvatarURL   *string                       `json:"avatar_url"`
	Rating      *int                          `json:"rating"`
	Provisional bool                          `json:"provisional"`
	Ratings     map[string]ModeRatingResponse `json:"ratings"`
	SkillLevel  *string                       `json:"skill_level"`
	NeedsRating bool                          `json:"needs_rating"`
	JoinedAt    time.Time                     `json:"joined_at"`
}

type UpdateProfileData struct {
	DisplayName *string `json:"display_name"`
	Country     *string `json:"country"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

type PublicProfileResponse struct {
	Username    string                        `json:"username"`
	DisplayName *string                       `json:"display_name"`
	Country     *string                       `json:"country"`
	Bio         *string                       `json:"bio"`
	AvatarURL   *string                       `json:"avatar_url"`
	Rating      *int                          `json:"rating"`
	Provisional bool                          `json:"provisional"`
	JoinedAt    time.Time                     `json:"joined_at"`
	Ratings     map[string]ModeRatingResponse `json:"ratings"`
}

type ModeRatingResponse struct {
	Rating      int  `json:"rating"`
	Provisional bool `json:"provisional"`
}

type RecentGameResponse struct {
	GameID           string     `json:"game_id"`
	OpponentUsername string     `json:"opponent_username"`
	Color            string     `json:"color"`
	Outcome          string     `json:"outcome"`
	EndReason        *string    `json:"end_reason"`
	EndedAt          *time.Time `json:"ended_at"`
}

type ModeStatsResponse struct {
	Mode         string               `json:"mode"`
	Rating       int                  `json:"rating"`
	Provisional  bool                 `json:"provisional"`
	BestRating   *int                 `json:"best_rating"`
	BestRatingAt *time.Time           `json:"best_rating_at"`
	GamesPlayed  int                  `json:"games_played"`
	Wins         int                  `json:"wins"`
	Losses       int                  `json:"losses"`
	Draws        int                  `json:"draws"`
	WinPercent   int                  `json:"win_percent"`
	RecentGames  []RecentGameResponse `json:"recent_games"`
}
