package worker

import (
	"log/slog"
	"os"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/hibiken/asynq"
)

func StartEmailServer(redisOpt asynq.RedisClientOpt, sender *email.Sender) {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return 2 * time.Minute
			},
		},
	)

	mux := asynq.NewServeMux()
	processor := NewEmailTaskProcessor(sender)
	mux.HandleFunc(TypeEmailRegistration, processor.ProcessTaskEmailRegistration)
	mux.HandleFunc(TypeEmailPasswordReset, processor.ProcessTaskEmailPasswordReset)

	go func() {
		if err := server.Run(mux); err != nil {
			slog.Error("could not run asynq server", "error", err)
			os.Exit(1)
		}
	}()
}
