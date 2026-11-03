package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
	"github.com/jackc/pgx/v5"
)

func RequireAdmin(userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if !ok {
			response.Abort(c, http.StatusForbidden, "admin access required")
			return
		}

		user, err := userRepo.FindUserByID(c.Request.Context(), userID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.Abort(c, http.StatusUnauthorized, "user not found")
			return
		}
		if err != nil {
			slog.Error("find admin user failed", "error", err)
			response.Abort(c, http.StatusInternalServerError, "internal server error")
			return
		}

		if user.IsBanned || user.Role != "admin" {
			response.Abort(c, http.StatusForbidden, "admin access required")
			return
		}

		c.Next()
	}
}
