package dto

type ChooseAvatarRequest struct {
	AvatarID string `json:"avatar_id" binding:"required"`
}

type DefaultAvatarResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
