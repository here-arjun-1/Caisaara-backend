package account

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=72"`
}
