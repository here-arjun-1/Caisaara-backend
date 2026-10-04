package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
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
			response.Abort(c, http.StatusUnauthorized, "authorization token is required")
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
			response.Abort(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if claims.IsGuest {
			if claims.GuestID == "" {
				response.Abort(c, http.StatusUnauthorized, "invalid guest token claims")
				return
			}

			c.Set("is_guest", true)
			c.Set("guest_id", claims.GuestID)
			c.Next()
			return
		}

		if claims.UserID <= 0 {
			response.Abort(c, http.StatusUnauthorized, "invalid token claims")
			return
		}

		if len(userRepo) > 0 && userRepo[0] != nil && claims.PasswordVersion > 0 {
			currentVersion, err := userRepo[0].GetPasswordVersion(c.Request.Context(), claims.UserID)
			if err != nil || claims.PasswordVersion != currentVersion {
				response.Abort(c, http.StatusUnauthorized, "invalid or revoked token")
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
			response.Abort(c, http.StatusForbidden, "feature requires a registered account")
			return
		}
		c.Next()
	}
}
