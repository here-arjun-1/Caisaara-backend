package dto

type SetRatingData struct {
	Level string `json:"level" binding:"required,oneof=new beginner intermediate advanced"`
}

type SetRatingResponse struct {
	Level  string `json:"level"`
	Rating int    `json:"rating"`
}
