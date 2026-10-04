package response

import "github.com/gin-gonic/gin"

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, status int, errMsg string) {
	c.JSON(status, Response{
		Success: false,
		Error:   errMsg,
	})
}

func Abort(c *gin.Context, status int, errMsg string) {
	c.AbortWithStatusJSON(status, Response{
		Success: false,
		Error:   errMsg,
	})
}
