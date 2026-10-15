package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

func GetUserID(c *gin.Context) (int64, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDValue.(int64)
	return userID, ok
}

func ClaimsFromToken(tokenString string) (*token.AccessTokenClaims, bool) {
	if tokenString == "" {
		return nil, false
	}

	claims, err := token.ValidateToken(tokenString)
	if err != nil || claims.IsGuest || claims.UserID == 0 {
		return nil, false
	}

	return claims, true
}
