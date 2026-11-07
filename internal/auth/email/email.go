package email

import (
	"fmt"
	"net/smtp"
	"time"

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

func (s *Sender) SendAccountExistsEmail(to string) error {
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

	subject := "You already have an account"

	body := "Hello,\n\nSomeone tried to create a new account with this email, but you already have one.\n\nIf it was you, just log in. If you forgot your password, use Forgot Password on the login screen.\n\nIf it wasn't you, you can ignore this email.\n"

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

func (s *Sender) SendAdminLoginAlertEmail(to string, ip string, at time.Time) error {
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

	subject := "New login to admin account"

	body := fmt.Sprintf(
		"Hello,\n\nThe admin account was just logged in.\n\nTime (UTC): %s\nIP address: %s\n\nIf this was not you, revoke all admin sessions and change the password right away.\n",
		at.Format("2006-01-02 15:04:05"),
		ip,
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
