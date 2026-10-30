package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/here-arjun-1/Caisaara-backend/internal/app"
	"github.com/here-arjun-1/Caisaara-backend/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	a, err := app.New(cfg)
	if err != nil {
		return err
	}
	defer a.Close()

	return a.Run()
}
