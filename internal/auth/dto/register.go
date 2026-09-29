package dto

type RegisterData struct {
	Username string `json:"username" binding:"required,username"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type VerifyRegistrationData struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,otp"`
}
type GuestLoginData struct {
	Username string `json:"username" binding:"required,username"`
}
