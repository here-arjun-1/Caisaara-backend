package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {

	userID, exists := c.Get("user_id")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "you accessed a protected route",
		"user_id": userID,
	})
}
