package token

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateGuestID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("guest_%s", hex.EncodeToString(bytes)), nil
}
