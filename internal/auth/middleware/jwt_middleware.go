package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
)

func JWTMiddleware(secret string, userRepo ...*repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Fields(authHeader)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			if cookieToken, err := c.Cookie("access_token"); err == nil && cookieToken != "" {
				tokenString = cookieToken
			}
		}


		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization token is required",
			})
			c.Abort()
			return
		}

		claims := &token.AccessTokenClaims{}
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

		if err != nil || !jwtToken.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		if claims.IsGuest {
			if claims.GuestID == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid guest token claims",
				})
				c.Abort()
				return
			}

			c.Set("is_guest", true)
			c.Set("guest_id", claims.GuestID)
			c.Next()
			return
		}

		if claims.UserID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			c.Abort()
			return
		}

		if len(userRepo) > 0 && userRepo[0] != nil && claims.PasswordVersion > 0 {
			currentVersion, err := userRepo[0].GetPasswordVersion(c.Request.Context(), claims.UserID)
			if err != nil || claims.PasswordVersion != currentVersion {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid or revoked token",
				})
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("is_guest", false)
		c.Next()
	}
}

func RequireRegisteredUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		isGuest, exists := c.Get("is_guest")
		if exists && isGuest.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "feature requires a registered account",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

