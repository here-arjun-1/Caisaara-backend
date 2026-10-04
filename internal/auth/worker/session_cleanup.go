package worker

import (
	"context"
	"log/slog"
	"time"
)

type SessionCleaner interface {
	CleanExpiredSessions(ctx context.Context) (int64, error)
}

func StartSessionCleanup(sessions SessionCleaner) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			deleted, err := sessions.CleanExpiredSessions(context.Background())
			if err != nil {
				slog.Error("failed to clean expired sessions", "error", err)
			} else if deleted > 0 {
				slog.Info("cleaned up expired sessions", "count", deleted)
			}
		}
	}()
}
