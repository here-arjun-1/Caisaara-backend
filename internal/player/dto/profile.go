package dto

import "time"

type MyProfileResponse struct {
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	Country     *string   `json:"country"`
	Bio         *string   `json:"bio"`
	AvatarURL   *string   `json:"avatar_url"`
	Rating      *int      `json:"rating"`
	SkillLevel  *string   `json:"skill_level"`
	NeedsRating bool      `json:"needs_rating"`
	JoinedAt    time.Time `json:"joined_at"`
}

type UpdateProfileData struct {
	DisplayName *string `json:"display_name"`
	Country     *string `json:"country"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

type PublicProfileResponse struct {
	Username    string    `json:"username"`
	DisplayName *string   `json:"display_name"`
	Country     *string   `json:"country"`
	Bio         *string   `json:"bio"`
	AvatarURL   *string   `json:"avatar_url"`
	Rating      *int      `json:"rating"`
	JoinedAt    time.Time `json:"joined_at"`
}
