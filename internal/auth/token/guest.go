package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const GuestTokenTTL = 24 * time.Hour

type GuestTokenClaims struct {
	GuestID  string `json:"guest_id"`
	Username string `json:"username"`
	IsGuest  bool   `json:"is_guest"`
	jwt.RegisteredClaims
}

func GenerateGuestID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("guest_%s", hex.EncodeToString(bytes)), nil
}

func GenerateGuestToken(guestID, username string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET is not set")
	}

	now := time.Now()
	claims := GuestTokenClaims{
		GuestID:  guestID,
		Username: username,
		IsGuest:  true,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   guestID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(GuestTokenTTL)),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}
