package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const refreshTokenCookieName = "refresh_token"
const refreshTokenMaxAge = 30 * 24 * 60 * 60

func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		refreshToken,
		refreshTokenMaxAge,
		"/",
		"",
		true,
		true,
	)
}

func ClearRefreshTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		"",
		-1,
		"/",
		"",
		true,
		true,
	)
}

func GetRefreshTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(refreshTokenCookieName)
}
