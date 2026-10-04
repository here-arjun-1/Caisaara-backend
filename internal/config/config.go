package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	SMTP        SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment variables")
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8050"),
		DatabaseURL: os.Getenv("dbURL"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnv("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
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
