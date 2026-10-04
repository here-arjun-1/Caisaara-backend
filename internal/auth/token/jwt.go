package token

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenTTL = 15 * time.Minute

type AccessTokenClaims struct {
	UserID          int64 `json:"user_id"`
	PasswordVersion int   `json:"password_version"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(secret string, userID int64, passwordVersion int) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		UserID:          userID,
		PasswordVersion: passwordVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString([]byte(secret))
}
