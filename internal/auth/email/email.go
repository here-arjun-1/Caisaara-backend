package email

import (
	"fmt"
	"net/smtp"

	"github.com/here-arjun-1/Caisaara-backend/internal/config"
)

type Sender struct {
	cfg config.SMTPConfig
}

func NewSender(cfg config.SMTPConfig) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) SendPasswordResetEmail(to string, code string) error {

	host := s.cfg.Host
	port := s.cfg.Port
	username := s.cfg.Username
	password := s.cfg.Password

	auth := smtp.PlainAuth(
		"",
		username,
		password,
		host,
	)

	subject := "Password Reset Verification Code"

	body := fmt.Sprintf(
		"Hello,\n\nYour password reset verification code is: %s\n\nThis code will expire in 15 minutes.\n",
		code,
	)

	message := []byte(
		"Subject: " + subject + "\r\n" +
			"From: " + username + "\r\n" +
			"To: " + to + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			body,
	)

	return smtp.SendMail(
		host+":"+port,
		auth,
		username,
		[]string{to},
		message,
	)
}

func (s *Sender) SendRegistrationEmail(to string, code string) error {
	host := s.cfg.Host
	port := s.cfg.Port
	username := s.cfg.Username
	password := s.cfg.Password

	auth := smtp.PlainAuth(
		"",
		username,
		password,
		host,
	)

	subject := "Registration Verification Code"

	body := fmt.Sprintf(
		"Hello,\n\nYour registration verification code is: %s\n\nThis code will expire in 15 minutes.\n",
		code,
	)

	message := []byte(
		"Subject: " + subject + "\r\n" +
			"From: " + username + "\r\n" +
			"To: " + to + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			body,
	)

	return smtp.SendMail(
		host+":"+port,
		auth,
		username,
		[]string{to},
		message,
	)
}
