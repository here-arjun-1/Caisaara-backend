package token

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenTTL = 15 * time.Minute
const GuestTokenTTL = 24 * time.Hour

type AccessTokenClaims struct {
	UserID          int64  `json:"user_id,omitempty"`
	GuestID         string `json:"guest_id,omitempty"`
	IsGuest         bool   `json:"is_guest"`
	PasswordVersion int    `json:"password_version,omitempty"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(secret string, userID int64, passwordVersion int) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		UserID:          userID,
		IsGuest:         false,
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

func GenerateGuestToken(secret string, guestID string) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		GuestID: guestID,
		IsGuest: true,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   guestID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(GuestTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString string) (*AccessTokenClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is not set")
	}

	claims := &AccessTokenClaims{}
	jwtToken, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !jwtToken.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
