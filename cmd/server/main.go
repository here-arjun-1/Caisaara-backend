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
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/worker"
	"github.com/here-arjun-1/Caisaara-backend/internal/player"
	"github.com/hibiken/asynq"
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

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	asynqRedisOpt := asynq.RedisClientOpt{Addr: redisURL}

	taskDistributor := asynq.NewClient(asynqRedisOpt)
	defer func() { _ = taskDistributor.Close() }()

	asynqServer := asynq.NewServer(
		asynqRedisOpt,
		asynq.Config{
			Concurrency: 10,
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return 2 * time.Minute
			},
		},
	)

	asynqMux := asynq.NewServeMux()
	emailProcessor := worker.NewEmailTaskProcessor()
	asynqMux.HandleFunc(worker.TypeEmailRegistration, emailProcessor.ProcessTaskEmailRegistration)
	asynqMux.HandleFunc(worker.TypeEmailPasswordReset, emailProcessor.ProcessTaskEmailPasswordReset)

	go func() {
		if err := asynqServer.Run(asynqMux); err != nil {
			log.Fatalf("could not run asynq server: %v", err)
		}
	}()

	userRepository := repository.NewUserRepository(conn)
	sessionRepository := repository.NewSessionRepository(conn)

	registerService := service.NewRegisterService(
		conn,
		userRepository,
		sessionRepository,
		redisClient,
		taskDistributor,
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
		taskDistributor,
	)
	passwordResetHandler := handler.NewPasswordResetHandler(
		passwordResetService,
	)

	ratingService := service.NewRatingService(userRepository)
	ratingHandler := handler.NewRatingHandler(ratingService)

	playerModule := player.NewModule(conn)

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

	loginLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	registerLimiter := middleware.NewTokenBucketLimiter(10, time.Hour)
	verifyRegistrationLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)
	guestLoginLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	forgotLimiter := middleware.NewTokenBucketLimiter(5, 15*time.Minute)
	verifyResetLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)
	resetPasswordLimiter := middleware.NewTokenBucketLimiter(5, 15*time.Minute)
	refreshLimiter := middleware.NewTokenBucketLimiter(30, time.Minute)
	logoutLimiter := middleware.NewTokenBucketLimiter(10, time.Minute)
	ratingLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	r.POST("/register", registerLimiter.Limit, registerHandler.Register)
	r.POST("/verify-registration", verifyRegistrationLimiter.Limit, registerHandler.VerifyRegistration)
	r.POST("/guest-login", guestLoginLimiter.Limit, registerHandler.GuestLogin)
	r.POST("/login", loginLimiter.Limit, loginHandler.Login)
	r.POST("/refresh", refreshLimiter.Limit, refreshHandler.Refresh)

	r.POST("/logout", logoutLimiter.Limit, logoutHandler.Logout)
	r.POST("/logout-all", logoutLimiter.Limit, logoutHandler.LogoutAll)

	r.POST("/forgot-password", forgotLimiter.Limit, passwordResetHandler.ForgotPassword)
	r.POST("/verify-reset-code", verifyResetLimiter.Limit, passwordResetHandler.VerifyCode)
	r.POST("/reset-password", resetPasswordLimiter.Limit, passwordResetHandler.ResetPassword)

	protected := r.Group("/api")

	protected.Use(middleware.JWTMiddleware())

	{
		protected.POST("/rating", ratingLimiter.Limit, ratingHandler.SetRating)
	}

	playerModule.RegisterRoutes(r, protected)

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

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			deleted, err := sessionRepository.CleanExpiredSessions(context.Background())
			if err != nil {
				log.Printf("failed to clean expired sessions: %v", err)
			} else if deleted > 0 {
				log.Printf("cleaned up %d expired sessions", deleted)
			}
		}
	}()

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
