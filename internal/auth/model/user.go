package model

type User struct {
	ID         int64
	Username   string
	Email      string
	Password   string
	Rating     int
	SkillLevel *string
}
