package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/hibiken/asynq"
)

const (
	TypeEmailRegistration  = "email:registration"
	TypeEmailPasswordReset = "email:password_reset"
)

type EmailPayload struct {
	To   string `json:"to"`
	Code string `json:"code"`
}

func NewEmailRegistrationTask(to, code string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailPayload{To: to, Code: code})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailRegistration, payload), nil
}

func NewEmailPasswordResetTask(to, code string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailPayload{To: to, Code: code})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailPasswordReset, payload), nil
}

type EmailTaskProcessor struct{}

func NewEmailTaskProcessor() *EmailTaskProcessor {
	return &EmailTaskProcessor{}
}

func (p *EmailTaskProcessor) ProcessTaskEmailRegistration(ctx context.Context, t *asynq.Task) error {
	var pld EmailPayload
	if err := json.Unmarshal(t.Payload(), &pld); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	err := email.SendRegistrationEmail(pld.To, pld.Code)
	if err != nil {
		return fmt.Errorf("could not send registration email: %w", err)
	}

	return nil
}

func (p *EmailTaskProcessor) ProcessTaskEmailPasswordReset(ctx context.Context, t *asynq.Task) error {
	var pld EmailPayload
	if err := json.Unmarshal(t.Payload(), &pld); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	err := email.SendPasswordResetEmail(pld.To, pld.Code)
	if err != nil {
		return fmt.Errorf("could not send password reset email: %w", err)
	}

	return nil
}
