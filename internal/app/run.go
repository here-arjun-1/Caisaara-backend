package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/analysis"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/worker"
	"github.com/here-arjun-1/Caisaara-backend/internal/notification"
	"github.com/here-arjun-1/Caisaara-backend/internal/tournament"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (a *App) Run() error {
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	asynqServer := worker.StartEmailServer(a.infra.asynqRedisOpt, email.NewSender(a.cfg.SMTP))
	analysisServer := analysis.StartAnalysisServer(a.infra.asynqRedisOpt, a.analysisProcessor, 2)

	var pushServer *asynq.Server
	if a.cfg.FirebaseCredentials != "" {
		pushSender, err := notification.NewPushSender(context.Background(), a.cfg.FirebaseCredentials)
		if err != nil {
			return fmt.Errorf("setup firebase: %w", err)
		}
		a.notificationService.SetTaskClient(a.infra.taskDistributor)
		pushServer = notification.StartPushServer(a.infra.asynqRedisOpt, notification.NewPushTaskProcessor(a.notificationRepository, pushSender))
	} else {
		slog.Info("FIREBASE_CREDENTIALS_BASE64 not set, push notifications are off")
	}

	worker.StartSessionCleanup(appCtx, a.authModule.SessionRepository, 1*time.Hour)
	a.matcher.Start(appCtx)
	tournament.StartTournamentCleanup(appCtx, a.tournamentRepo, 24*time.Hour)
	a.gameService.StartTimeoutChecker(appCtx, 1*time.Second)
	a.ratingService.StartMissedRatingsChecker(appCtx, 1*time.Minute)
	a.wsHandler.StartActiveAuthChecker(appCtx, 5*time.Second)

	srv := newHTTPServer(":"+a.cfg.Port, a.router)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	metricsSrv := newHTTPServer(":9100", promhttp.Handler())

	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	slog.Info("server started", "port", a.cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	appCancel()

	a.wsHandler.Hub.CloseAll()

	if asynqServer != nil {
		asynqServer.Stop()
	}
	if analysisServer != nil {
		analysisServer.Stop()
	}
	if pushServer != nil {
		pushServer.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = metricsSrv.Shutdown(ctx)

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	slog.Info("server exited")
	return nil
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
