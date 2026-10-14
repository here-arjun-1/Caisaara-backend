package worker

import (
	"context"
	"log/slog"
	"time"
)

type SessionCleaner interface {
	CleanExpiredSessions(ctx context.Context) (int64, error)
}

func StartSessionCleanup(ctx context.Context, sessions SessionCleaner, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deleted, err := sessions.CleanExpiredSessions(ctx)
				if err != nil {
					slog.Error("failed to clean expired sessions", "error", err)
				} else if deleted > 0 {
					slog.Info("cleaned up expired sessions", "count", deleted)
				}
			}
		}
	}()
}
