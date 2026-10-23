package dto

import "time"

type MyProfileResponse struct {
	Username     string                        `json:"username"`
	Email        string                        `json:"email"`
	DisplayName  *string                       `json:"display_name"`
	Country      *string                       `json:"country"`
	Bio          *string                       `json:"bio"`
	AvatarURL    *string                       `json:"avatar_url"`
	Rating       *int                          `json:"rating"`
	Provisional  bool                          `json:"provisional"`
	Ratings      map[string]ModeRatingResponse `json:"ratings"`
	SkillLevel   *string                       `json:"skill_level"`
	NeedsRating  bool                          `json:"needs_rating"`
	FriendsCount int                           `json:"friends_count"`
	JoinedAt     time.Time                     `json:"joined_at"`
}

type UpdateProfileData struct {
	DisplayName *string `json:"display_name"`
	Country     *string `json:"country"`
	Bio         *string `json:"bio"`
}

type PublicProfileResponse struct {
	UserID      int64                         `json:"user_id"`
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
	Wins        int  `json:"wins"`
	Losses      int  `json:"losses"`
	Draws       int  `json:"draws"`
}

type RecentGameResponse struct {
	GameID           string     `json:"game_id"`
	OpponentUsername string     `json:"opponent_username"`
	Color            string     `json:"color"`
	Outcome          string     `json:"outcome"`
	EndReason        *string    `json:"end_reason"`
	EndedAt          *time.Time `json:"ended_at"`
}

type RecordResponse struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
	Draws  int `json:"draws"`
}

type RatingPointResponse struct {
	Date   string `json:"date"`
	Rating int    `json:"rating"`
}

type ModeStatsResponse struct {
	Mode           string                `json:"mode"`
	Range          string                `json:"range"`
	Rating         int                   `json:"rating"`
	Provisional    bool                  `json:"provisional"`
	BestRating     *int                  `json:"best_rating"`
	BestRatingAt   *time.Time            `json:"best_rating_at"`
	LowestRating   *int                  `json:"lowest_rating"`
	LowestRatingAt *time.Time            `json:"lowest_rating_at"`
	StartRating    int                   `json:"start_rating"`
	GamesPlayed    int                   `json:"games_played"`
	Wins           int                   `json:"wins"`
	Losses         int                   `json:"losses"`
	Draws          int                   `json:"draws"`
	WinPercent     int                   `json:"win_percent"`
	AsWhite        RecordResponse        `json:"as_white"`
	AsBlack        RecordResponse        `json:"as_black"`
	CurrentStreak  int                   `json:"current_streak"`
	BestStreak     int                   `json:"best_streak"`
	RatingHistory  []RatingPointResponse `json:"rating_history"`
	RecentGames    []RecentGameResponse  `json:"recent_games"`
}

type PlayerSearchResult struct {
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Rating      *int    `json:"rating"`
}
