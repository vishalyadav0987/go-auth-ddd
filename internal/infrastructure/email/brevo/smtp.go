package brevo

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
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
		strings.ContainsAny(subject, "\r\n") ||
		strings.ContainsAny(s.from, "\r\n") {
		return fmt.Errorf("invalid email header")
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

	// Step 1: Check whether the server supports STARTTLS.
	ok, _ := client.Extension("STARTTLS")

	if !ok {
		return domain.NewProviderError(
			domain.ProviderErrorPermanent,
			"SMTP_STARTTLS_UNAVAILABLE",
			"SMTP server does not support STARTTLS",
			nil,
		)
	}

	if !ok {
		return domain.NewProviderError(
			domain.ProviderErrorPermanent,
			"SMTP_STARTTLS_UNAVAILABLE",
			"SMTP server does not support STARTTLS",
			nil,
		)
	}

	// Step 2: Upgrade the connection to TLS BEFORE authentication.
	tlsConfig := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return domain.NewProviderError(
			domain.ProviderErrorTransient,
			"SMTP_TLS_FAILED",
			"failed to establish SMTP TLS connection",
			err,
		)
	}

	// Step 3: Authenticate only after TLS is established.
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	if err := client.Auth(auth); err != nil {
		return classifySMTPError("AUTH", err)
	}

	// Step 4: Set sender.
	if err := client.Mail(s.from); err != nil {
		return classifySMTPError("MAIL FROM", err)
	}

	// Step 5: Set recipient.
	if err := client.Rcpt(to); err != nil {
		return classifySMTPError("RCPT TO", err)
	}

	// Step 6: Start message transmission.
	writer, err := client.Data()
	if err != nil {
		return classifySMTPError("DATA", err)
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
		return classifySMTPError("WRITE MESSAGE", err)
	}

	// Step 7: Close DATA writer.
	// A successful Close means the SMTP server accepted the message.
	if err := writer.Close(); err != nil {
		return classifySMTPError("FINISH MESSAGE", err)
	}

	// Do not report failure just because QUIT fails.
	// The server has already accepted the message after successful DATA Close.
	_ = client.Quit()

	return nil
}

func classifySMTPError(
	operation string,
	err error,
) error {
	if err == nil {
		return nil
	}

	var smtpErr *textproto.Error

	if errors.As(err, &smtpErr) {
		switch {
		case smtpErr.Code >= 400 && smtpErr.Code < 500:
			return domain.NewProviderError(
				domain.ProviderErrorTransient,
				"SMTP_TEMPORARY",
				"temporary SMTP rejection during "+operation,
				err,
			)

		case smtpErr.Code >= 500 && smtpErr.Code < 600:
			return domain.NewProviderError(
				domain.ProviderErrorPermanent,
				"SMTP_PERMANENT",
				"permanent SMTP rejection during "+operation,
				err,
			)
		}
	}

	var netErr net.Error

	if errors.As(err, &netErr) && netErr.Timeout() {
		return domain.NewProviderError(
			domain.ProviderErrorAmbiguous,
			"SMTP_TIMEOUT",
			"SMTP operation timed out during "+operation,
			err,
		)
	}

	return domain.NewProviderError(
		domain.ProviderErrorAmbiguous,
		"SMTP_UNKNOWN",
		"SMTP outcome is uncertain during "+operation,
		err,
	)
}

var _ domain.EmailProvider = (*SMTPEmailProvider)(nil)
