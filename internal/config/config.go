package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	SMTP          SMTPConfig
	S3            S3Config
	StockfishPath string
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

type S3Config struct {
	Region    string
	Bucket    string
	PublicURL string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using system environment variables")
	}

	cfg := &Config{
		Port:          getEnv("PORT", "8050"),
		DatabaseURL:   os.Getenv("dbURL"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		StockfishPath: getEnv("STOCKFISH_PATH", "stockfish"),
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnv("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
		},
		S3: S3Config{
			Region:    os.Getenv("AWS_REGION"),
			Bucket:    os.Getenv("S3_BUCKET"),
			PublicURL: strings.TrimSuffix(os.Getenv("S3_PUBLIC_URL"), "/"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "dbURL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.SMTP.Username == "" {
		missing = append(missing, "SMTP_USERNAME")
	}
	if c.SMTP.Password == "" {
		missing = append(missing, "SMTP_PASSWORD")
	}
	if c.S3.Region == "" {
		missing = append(missing, "AWS_REGION")
	}
	if c.S3.Bucket == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if c.S3.PublicURL == "" {
		missing = append(missing, "S3_PUBLIC_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
