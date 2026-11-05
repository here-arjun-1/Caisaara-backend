package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/here-arjun-1/Caisaara-backend/internal/auth/email"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/repository"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
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
	return asynq.NewTask(TypeEmailRegistration, payload, asynq.Queue("email")), nil
}

func NewEmailPasswordResetTask(to, code string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailPayload{To: to, Code: code})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeEmailPasswordReset, payload, asynq.Queue("email")), nil
}

type EmailTaskProcessor struct {
	sender         *email.Sender
	userRepository *repository.UserRepository
}

func NewEmailTaskProcessor(sender *email.Sender, userRepository *repository.UserRepository) *EmailTaskProcessor {
	return &EmailTaskProcessor{sender: sender, userRepository: userRepository}
}

func (p *EmailTaskProcessor) ProcessTaskEmailRegistration(ctx context.Context, t *asynq.Task) error {
	var pld EmailPayload
	if err := json.Unmarshal(t.Payload(), &pld); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %w: %w", err, asynq.SkipRetry)
	}

	_, err := p.userRepository.FindUserByEmail(ctx, pld.To)
	if err == nil {
		err = p.sender.SendAccountExistsEmail(pld.To)
		if err != nil {
			return fmt.Errorf("could not send account exists email: %w", err)
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("find user by email failed: %w", err)
	}

	err = p.sender.SendRegistrationEmail(pld.To, pld.Code)
	if err != nil {
		return fmt.Errorf("could not send registration email: %w", err)
	}

	return nil
}

func (p *EmailTaskProcessor) ProcessTaskEmailPasswordReset(ctx context.Context, t *asynq.Task) error {
	var pld EmailPayload
	if err := json.Unmarshal(t.Payload(), &pld); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %w: %w", err, asynq.SkipRetry)
	}

	_, err := p.userRepository.FindUserByEmail(ctx, pld.To)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user by email failed: %w", err)
	}

	err = p.sender.SendPasswordResetEmail(pld.To, pld.Code)
	if err != nil {
		return fmt.Errorf("could not send password reset email: %w", err)
	}

	return nil
}
