package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/config"
)

//go:embed templates/*.html
var templateFiles embed.FS

var codeEmailTemplate = template.Must(template.ParseFS(templateFiles, "templates/code_email.html"))

type CodeEmailData struct {
	Subject    string
	Eyebrow    string
	TitleLine1 string
	TitleLine2 string
	Username   string
	Intro      string
	CodeLabel  string
	Code       string
	ExpiresIn  string
	Hint       string
	NoteTitle  string
	NoteText   string
}

type Sender struct {
	cfg config.SMTPConfig
}

func NewSender(cfg config.SMTPConfig) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) SendPasswordResetEmail(to string, username string, code string) error {
	data := CodeEmailData{
		Subject:    "Your Mosaic password recovery code",
		Eyebrow:    "Password recovery",
		TitleLine1: "A fresh start.",
		TitleLine2: "Same great game.",
		Username:   username,
		Intro:      "Forgot your password? Enter this recovery code in Mosaic to set a new password.",
		CodeLabel:  "Password recovery code",
		Code:       splitCode(code),
		ExpiresIn:  "5 minutes",
		Hint:       "Go back to the recovery screen and enter the code above.",
		NoteTitle:  "Didn't request a reset?",
		NoteText:   "You can safely ignore this email. Your password will stay the same unless you complete the reset.",
	}

	textBody := fmt.Sprintf(
		"Hello,\n\nYour password reset verification code is: %s\n\nThis code will expire in 5 minutes.\n",
		code,
	)

	return s.sendHTMLEmail(to, data, textBody)
}

func (s *Sender) sendHTMLEmail(to string, data CodeEmailData, textBody string) error {
	var htmlBody bytes.Buffer
	if err := codeEmailTemplate.Execute(&htmlBody, data); err != nil {
		return fmt.Errorf("render email template: %w", err)
	}

	boundary := "mosaic-email-boundary"
	username := s.cfg.Username

	message := []byte(
		"Subject: " + data.Subject + "\r\n" +
			"From: " + username + "\r\n" +
			"To: " + to + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n" +
			"\r\n" +
			"--" + boundary + "\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			textBody + "\r\n" +
			"--" + boundary + "\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			htmlBody.String() + "\r\n" +
			"--" + boundary + "--\r\n",
	)

	auth := smtp.PlainAuth("", username, s.cfg.Password, s.cfg.Host)

	return smtp.SendMail(s.cfg.Host+":"+s.cfg.Port, auth, username, []string{to}, message)
}

func splitCode(code string) string {
	if len(code) != 6 {
		return code
	}
	return code[:3] + " " + code[3:]
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

func (s *Sender) SendRegistrationEmail(to string, username string, code string) error {
	data := CodeEmailData{
		Subject:    "Your Mosaic verification code",
		Eyebrow:    "Account verification",
		TitleLine1: "One code.",
		TitleLine2: "Your next move.",
		Username:   username,
		Intro:      "You're one move away from Mosaic. Enter this code in the app to verify your email and finish creating your account.",
		CodeLabel:  "Email verification code",
		Code:       splitCode(code),
		ExpiresIn:  "5 minutes",
		Hint:       "Go back to the verification screen and enter the code above.",
		NoteTitle:  "Keep this code private.",
		NoteText:   "Never share this code, even with someone claiming to be from Mosaic. Didn't create an account? You can ignore this email.",
	}

	textBody := fmt.Sprintf(
		"Hello,\n\nYour registration verification code is: %s\n\nThis code will expire in 5 minutes.\n",
		code,
	)

	return s.sendHTMLEmail(to, data, textBody)
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
