package mailer

import (
	"bytes"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"os/exec"
	"strings"
)

type Mailer interface {
	Send(to, from, subject, body string) error
}

// SystemMailer submits messages to the local sendmail-compatible mailer.
type SystemMailer struct {
	command string
}

func NewSystemMailer(command string) *SystemMailer {
	if command == "" {
		command = "sendmail"
	}
	return &SystemMailer{command: command}
}

func (m *SystemMailer) Send(to, from, subject, body string) error {
	message, _, _, err := buildMessage(to, from, subject, body)
	if err != nil {
		return err
	}
	command := exec.Command(m.command, "-t", "-i")
	command.Stdin = bytes.NewReader(message)
	if err := command.Run(); err != nil {
		return fmt.Errorf("local mail submission failed: %w", err)
	}
	return nil
}

type SMTPMailer struct {
	host string
	port string
	user string
	pass string
}

func NewSMTPMailer(host, port, user, pass string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, user: user, pass: pass}
}

func (m *SMTPMailer) Send(to, from, subject, body string) error {
	message, recipient, sender, err := buildMessage(to, from, subject, body)
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if m.user != "" || m.pass != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	address := fmt.Sprintf("%s:%s", m.host, m.port)
	if err := smtp.SendMail(address, auth, sender.Address, []string{recipient.Address}, message); err != nil {
		return fmt.Errorf("SMTP mail submission failed: %w", err)
	}
	return nil
}

func buildMessage(to, from, subject, body string) ([]byte, *mail.Address, *mail.Address, error) {
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(from, "\r\n") {
		return nil, nil, nil, fmt.Errorf("invalid email address")
	}
	if strings.ContainsAny(subject, "\r\n") {
		return nil, nil, nil, fmt.Errorf("invalid email subject")
	}
	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address == "" || recipient.Address != to {
		return nil, nil, nil, fmt.Errorf("invalid recipient email")
	}
	sender, err := mail.ParseAddress(from)
	if err != nil || sender.Address == "" || sender.Address != from {
		return nil, nil, nil, fmt.Errorf("invalid sender email")
	}

	message := fmt.Sprintf(
		"To: %s\r\nFrom: %s\r\nReply-To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		recipient.String(),
		sender.String(),
		sender.String(),
		mime.QEncoding.Encode("UTF-8", subject),
		body,
	)
	return []byte(message), recipient, sender, nil
}
