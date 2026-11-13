package brevo

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type SMTPEmailProvider struct {
	from     string
	username string
	password string
	host     string
	port     string
}

func NewSMTPEmailProvider() (*SMTPEmailProvider, error) {
	provider := &SMTPEmailProvider{
		from:     os.Getenv("EMAIL_ID"),
		username: os.Getenv("BREVO_EMAIL_API"),
		password: os.Getenv("PASSWORD"),
		host:     os.Getenv("HOST"),
		port:     os.Getenv("EMAIL_SERVICE_PORT"),
	}

	if provider.from == "" ||
		provider.username == "" ||
		provider.password == "" ||
		provider.host == "" ||
		provider.port == "" {
		return nil, fmt.Errorf("missing SMTP configuration")
	}

	return provider, nil
}

func (s *SMTPEmailProvider) Send(
	ctx context.Context,
	to string,
	subject string,
	htmlBody string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if strings.ContainsAny(to, "\r\n") ||
		strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid email recipient or subject")
	}

	address := net.JoinHostPort(s.host, s.port)

	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}

	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		s.from,
		to,
		subject,
		htmlBody,
	)

	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write email message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}

	return nil
}

var _ notification.EmailProvider = (*SMTPEmailProvider)(nil)
