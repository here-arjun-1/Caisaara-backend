package dto

type RegisterData struct {
	Username string `json:"username" binding:"required,username"`
	Email    string `json:"email" binding:"required,strictemail"`
	Password string `json:"password" binding:"required,password"`
}

type VerifyRegistrationData struct {
	Email string `json:"email" binding:"required,strictemail"`
	Code  string `json:"code" binding:"required,otp"`
}
