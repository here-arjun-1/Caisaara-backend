package validation

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	usernameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	otpRegex         = regexp.MustCompile(`^\d{6}$`)
	localPartRegex   = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9._%+\-]*[a-zA-Z0-9_])?$`)
	domainLabelRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9\-]*[a-zA-Z0-9])?$`)
	tldRegex         = regexp.MustCompile(`^[a-zA-Z]{2,63}$`)
)

func Username(f validator.FieldLevel) bool {
	return usernameRegex.MatchString(f.Field().String())
}

func OTP(f validator.FieldLevel) bool {
	return otpRegex.MatchString(f.Field().String())
}

func Email(f validator.FieldLevel) bool {
	email := f.Field().String()
	if len(email) > 254 {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]

	if len(local) == 0 || len(local) > 64 {
		return false
	}
	if strings.Contains(local, "..") || !localPartRegex.MatchString(local) {
		return false
	}

	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) > 63 || !domainLabelRegex.MatchString(label) {
			return false
		}
	}

	return tldRegex.MatchString(labels[len(labels)-1])
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
