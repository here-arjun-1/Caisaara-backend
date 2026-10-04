package dto

type RegisterData struct {
	Username string `json:"username" binding:"required,username"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,password"`
}

type VerifyRegistrationData struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,otp"`
}

type VerifyRegistrationResponse struct {
	NeedsRating bool `json:"needs_rating"`
}

type GuestLoginResponse struct {
	GuestID string `json:"guest_id"`
}
