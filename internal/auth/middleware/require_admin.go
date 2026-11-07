package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/token"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
	"github.com/jackc/pgx/v5"
)

const RoleAdmin = "admin"

func accessTokenFromRequest(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Fields(authHeader)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	cookieToken, err := c.Cookie("access_token")
	if err == nil {
		return cookieToken
	}

	return ""
}

func RequireAdmin(secret string, userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := accessTokenFromRequest(c)
		if tokenString == "" {
			response.Abort(c, http.StatusUnauthorized, "authorization token is required")
			return
		}

		claims, err := token.ValidateToken(tokenString, secret)
		if err != nil {
			response.Abort(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if claims.IsGuest || claims.UserID <= 0 {
			response.Abort(c, http.StatusForbidden, "admin access required")
			return
		}

		user, err := userRepo.FindUserByID(c.Request.Context(), claims.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.Abort(c, http.StatusUnauthorized, "invalid or revoked token")
			return
		}
		if err != nil {
			slog.Error("find admin user failed", "error", err)
			response.Abort(c, http.StatusInternalServerError, "internal server error")
			return
		}

		if claims.PasswordVersion != user.PasswordVersion {
			response.Abort(c, http.StatusUnauthorized, "invalid or revoked token")
			return
		}

		if user.IsBanned || user.Role != RoleAdmin {
			response.Abort(c, http.StatusForbidden, "admin access required")
			return
		}

		c.Set("user_id", user.ID)
		c.Set("is_guest", false)
		c.Next()
	}
}
