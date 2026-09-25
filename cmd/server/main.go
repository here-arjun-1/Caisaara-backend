package main

import (
	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/handler"
)

func main() {
	r := gin.Default()
	registerHandler := handler.NewRegisterHandler()
	r.POST("/register", registerHandler.Register)
	r.Run(":8050")
}
