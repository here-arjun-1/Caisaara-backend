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
