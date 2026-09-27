package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/database"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
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

	defer func() {
		if err := conn.Close(context.Background()); err != nil {
			log.Printf(
				"failed to close database connection: %v",
				err,
			)
		}
	}()
	userRepository := repository.NewUserRepository(conn)
	sessionRepository := repository.NewSessionRepository(conn)

	registerService := service.NewRegisterService(
		userRepository,
		sessionRepository,
	)

	registerHandler := handler.NewRegisterHandler(
		registerService,
	)
	verifyEmailService := service.NewVerifyEmailService(
		userRepository,
	)

	verifyEmailHandler := handler.NewVerifyEmailHandler(
		verifyEmailService,
	)
	loginService := service.NewLoginService(
		userRepository,
		sessionRepository,
	)

	loginHandler := handler.NewLoginHandler(
		loginService,
	)
	refreshService := service.NewRefreshService(
		sessionRepository,
	)

	refreshHandler := handler.NewRefreshHandler(
		refreshService,
	)

	logoutService := service.NewLogoutService(
		sessionRepository,
	)

	logoutHandler := handler.NewLogoutHandler(
		logoutService,
	)

	r := gin.Default()

	r.POST("/register", registerHandler.Register)
	r.POST("/login", loginHandler.Login)
	r.POST("/refresh", refreshHandler.Refresh)
	r.GET("/verify-email", verifyEmailHandler.VerifyEmail)

	r.POST("/logout", logoutHandler.Logout)
	r.POST("/logout-all", logoutHandler.LogoutAll)
	protected := r.Group("/api")

	protected.Use(middleware.JWTMiddleware())

	{
		protected.GET("/profile", handler.GetProfile)
	}
	if err := r.Run(":8050"); err != nil {
		log.Printf(
			"server failed to start: %v",
			err,
		)
	}
}
