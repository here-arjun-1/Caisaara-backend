package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const refreshTokenCookieName = "refresh_token"
const refreshTokenMaxAge = 30 * 24 * 60 * 60

const accessTokenCookieName = "access_token"
const accessTokenMaxAge = 15 * 60

const guestTokenCookieName = "guest_token"
const guestTokenMaxAge = 24 * 60 * 60

func isSecure(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}

func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshTokenCookieName,
		refreshToken,
		refreshTokenMaxAge,
		"/",
		"",
		isSecure(c),
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
		isSecure(c),
		true,
	)
}

func GetRefreshTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(refreshTokenCookieName)
}

func SetAccessTokenCookie(c *gin.Context, accessToken string, maxAge ...int) {
	age := accessTokenMaxAge
	if len(maxAge) > 0 {
		age = maxAge[0]
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		accessTokenCookieName,
		accessToken,
		age,
		"/",
		"",
		isSecure(c),
		true,
	)
}

func ClearAccessTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		accessTokenCookieName,
		"",
		-1,
		"/",
		"",
		isSecure(c),
		true,
	)
}

func GetAccessTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(accessTokenCookieName)
}

func SetGuestTokenCookie(c *gin.Context, guestToken string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		guestTokenCookieName,
		guestToken,
		guestTokenMaxAge,
		"/",
		"",
		isSecure(c),
		true,
	)
}

func ClearGuestTokenCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		guestTokenCookieName,
		"",
		-1,
		"/",
		"",
		isSecure(c),
		true,
	)
}

func GetGuestTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(guestTokenCookieName)
}
