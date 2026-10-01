package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/database"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/validation"
	"github.com/joho/godotenv"
)

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
func run() error {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}
	conn, err := database.ConnectDB()

	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer conn.Close()

	redisClient, err := database.ConnectRedis()
	if err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	defer func() { _ = redisClient.Close() }()

	userRepository := repository.NewUserRepository(conn)
	sessionRepository := repository.NewSessionRepository(conn)

	registerService := service.NewRegisterService(
		userRepository,
		sessionRepository,
		redisClient,
	)

	registerHandler := handler.NewRegisterHandler(
		registerService,
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

	passwordResetRepository := repository.NewPasswordResetRepository(conn)
	passwordResetService := service.NewPasswordResetService(
		userRepository,
		passwordResetRepository,
		sessionRepository,
		redisClient,
	)
	passwordResetHandler := handler.NewPasswordResetHandler(
		passwordResetService,
	)

	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return errors.New("failed to get validator engine")
	}
	if err := v.RegisterValidation("username", validation.Username); err != nil {
		return fmt.Errorf("register username validator: %w", err)
	}
	if err := v.RegisterValidation("otp", validation.OTP); err != nil {
		return fmt.Errorf("register otp validator: %w", err)
	}
	if err := v.RegisterValidation("password", validation.Password); err != nil {
		return fmt.Errorf("register password validator: %w", err)
	}

	r := gin.Default()

	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		return fmt.Errorf("set trusted proxies: %w", err)
	}

	loginLimiter := middleware.NewFixedWindowLimiter(30, time.Minute)
	registerLimiter := middleware.NewFixedWindowLimiter(30, 10*time.Minute)
	forgotLimiter := middleware.NewFixedWindowLimiter(30, 15*time.Minute)
	otpLimiter := middleware.NewFixedWindowLimiter(30, time.Minute)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	r.POST("/register", registerLimiter.Limit, registerHandler.Register)
	r.POST("/verify-registration", otpLimiter.Limit, registerHandler.VerifyRegistration)
	r.POST("/guest-login", loginLimiter.Limit, registerHandler.GuestLogin)
	r.POST("/login", loginLimiter.Limit, loginHandler.Login)
	r.POST("/refresh", refreshHandler.Refresh)

	r.POST("/logout", logoutHandler.Logout)
	r.POST("/logout-all", logoutHandler.LogoutAll)

	r.POST("/auth/forgot-password", forgotLimiter.Limit, passwordResetHandler.ForgotPassword)
	r.POST("/auth/verify-reset-code", otpLimiter.Limit, passwordResetHandler.VerifyCode)
	r.POST("/auth/reset-password", passwordResetHandler.ResetPassword)

	protected := r.Group("/api")

	protected.Use(middleware.JWTMiddleware())

	{
		protected.GET("/profile", handler.GetProfile)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8050"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	log.Printf("server started on port %s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("server exited")
	return nil
}
