package notification

import (
	"context"
	"encoding/base64"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type PushSender struct {
	client *messaging.Client
}

func NewPushSender(ctx context.Context, credentialsBase64 string) (*PushSender, error) {
	credentials, err := base64.StdEncoding.DecodeString(credentialsBase64)
	if err != nil {
		return nil, fmt.Errorf("decode firebase credentials: %w", err)
	}

	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentialsJSON(option.ServiceAccount, credentials))
	if err != nil {
		return nil, fmt.Errorf("create firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("create firebase messaging client: %w", err)
	}

	return &PushSender{client: client}, nil
}

func (p *PushSender) Send(ctx context.Context, token string, payload PushPayload) error {
	data := map[string]string{
		"type":            payload.Type,
		"notification_id": fmt.Sprint(payload.NotificationID),
	}
	for key, value := range payload.Data {
		data[key] = fmt.Sprint(value)
	}

	message := &messaging.Message{
		Token: token, //nolint:staticcheck
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
	}

	_, err := p.client.Send(ctx, message)
	if messaging.IsUnregistered(err) {
		return ErrDeadToken
	}
	return err
}
