package validation

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	otpRegex      = regexp.MustCompile(`^\d{6}$`)

	passwordUpperRegex   = regexp.MustCompile(`[A-Z]`)
	passwordLowerRegex   = regexp.MustCompile(`[a-z]`)
	passwordNumberRegex  = regexp.MustCompile(`[0-9]`)
	passwordSpecialRegex = regexp.MustCompile(`[!@#\$%\^&\*\(\)_\+\-\=\[\]\{\};':"\\\|,\.<>\/\?]`)
	passwordAllowedRegex = regexp.MustCompile(`^[a-zA-Z0-9!@#\$%\^&\*\(\)_\+\-\=\[\]\{\};':"\\\|,\.<>\/\?]*$`)
)

func Register() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return errors.New("failed to get validator engine")
	}
	if err := v.RegisterValidation("username", Username); err != nil {
		return fmt.Errorf("register username validator: %w", err)
	}
	if err := v.RegisterValidation("otp", OTP); err != nil {
		return fmt.Errorf("register otp validator: %w", err)
	}
	if err := v.RegisterValidation("password", Password); err != nil {
		return fmt.Errorf("register password validator: %w", err)
	}
	return nil
}

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
	hasUpper := passwordUpperRegex.MatchString(password)
	hasLower := passwordLowerRegex.MatchString(password)
	hasNumber := passwordNumberRegex.MatchString(password)
	hasSpecial := passwordSpecialRegex.MatchString(password)
	noEmojis := passwordAllowedRegex.MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial && noEmojis
}
