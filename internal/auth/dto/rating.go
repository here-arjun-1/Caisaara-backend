package dto

type SetRatingData struct {
	Level string `json:"level" binding:"required,oneof=new beginner intermediate advanced"`
}
