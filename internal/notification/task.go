package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hibiken/asynq"
)

const TypePushNotification = "notification:push"

type PushPayload struct {
	UserID         int64          `json:"user_id"`
	NotificationID int64          `json:"notification_id"`
	Type           string         `json:"type"`
	Title          string         `json:"title"`
	Body           string         `json:"body"`
	Data           map[string]any `json:"data"`
}

func NewPushTask(n *Notification) (*asynq.Task, error) {
	payload, err := json.Marshal(PushPayload{
		UserID:         n.UserID,
		NotificationID: n.ID,
		Type:           n.Type,
		Title:          n.Title,
		Body:           n.Body,
		Data:           n.Data,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePushNotification, payload, asynq.Queue("notifications"), asynq.MaxRetry(3)), nil
}

type PushTaskProcessor struct {
	repository *Repository
	sender     *PushSender
}

func NewPushTaskProcessor(repository *Repository, sender *PushSender) *PushTaskProcessor {
	return &PushTaskProcessor{
		repository: repository,
		sender:     sender,
	}
}

func (p *PushTaskProcessor) ProcessTaskPush(ctx context.Context, t *asynq.Task) error {
	var payload PushPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %w: %w", err, asynq.SkipRetry)
	}

	tokens, err := p.repository.FindDeviceTokens(ctx, payload.UserID)
	if err != nil {
		return err
	}

	sent := 0
	var lastErr error
	for _, token := range tokens {
		err := p.sender.Send(ctx, token, payload)
		if err == nil {
			sent++
			continue
		}

		if errors.Is(err, ErrDeadToken) {
			_ = p.repository.DeleteDeadToken(ctx, token)
			continue
		}

		slog.ErrorContext(ctx, "send push notification failed", "user_id", payload.UserID, "notification_id", payload.NotificationID, "error", err)
		lastErr = err
	}

	if sent == 0 && lastErr != nil {
		return lastErr
	}
	return nil
}

func StartPushServer(redisOpt asynq.RedisClientOpt, processor *PushTaskProcessor) *asynq.Server {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"notifications": 1,
			},
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return 30 * time.Second
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(TypePushNotification, processor.ProcessTaskPush)

	if err := server.Start(mux); err != nil {
		slog.Error("could not start push notification server", "error", err)
		os.Exit(1)
	}
	return server
}
