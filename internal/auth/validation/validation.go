package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	otpRegex      = regexp.MustCompile(`^\d{6}$`)
)

func Username(f validator.FieldLevel) bool {
	return usernameRegex.MatchString(f.Field().String())
}

func OTP(f validator.FieldLevel) bool {
	return otpRegex.MatchString(f.Field().String())
}

func Password(f validator.FieldLevel) bool {
	password := f.Field().String()
	if len(password) < 8 || len(password) > 16 {
		return false
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#\$%\^&\*\(\)_\+\-\=\[\]\{\};':"\\\|,\.<>\/\?]`).MatchString(password)
	noEmojis := regexp.MustCompile(`^[a-zA-Z0-9!@#\$%\^&\*\(\)_\+\-\=\[\]\{\};':"\\\|,\.<>\/\?]*$`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial && noEmojis
}
