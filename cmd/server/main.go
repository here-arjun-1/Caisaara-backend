package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/database"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env")
	}
	conn, err := database.ConnectDB()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	fmt.Println("db connected")
	defer conn.Close(context.Background())

	userRepository := repository.NewUserRepository(conn)
	registerService := service.NewRegisterService(userRepository)
	registerHandler := handler.NewRegisterHandler(registerService)
	loginService := service.NewLoginService(userRepository)
	loginHandler := handler.NewLoginHandler(loginService)

	r := gin.Default()
	r.POST("/register", registerHandler.Register)
	r.POST("/login", loginHandler.Login)
	r.Run(":8050")
}
