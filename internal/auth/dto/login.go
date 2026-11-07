package dto

type LoginData struct {
	Username string `json:"username" binding:"required_without=Email"`
	Email    string `json:"email" binding:"required_without=Username"`
	Password string `json:"password" binding:"required"`
	IP       string `json:"-"`
}

type LoginResponse struct {
	Username    string `json:"username"`
	NeedsRating bool   `json:"needs_rating"`
}
