package notification

import (
	"context"
	"errors"
	"strings"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type SendEmailInput struct {
	To       string
	Subject  string
	HTMLBody string
}

type EmailService struct {
	provider domain.EmailProvider
}

func NewEmailService(provider domain.EmailProvider) *EmailService {
	return &EmailService{
		provider: provider,
	}
}

func (s *EmailService) Send(
	ctx context.Context,
	input SendEmailInput,
) error {
	if s.provider == nil {
		return errors.New("email provider is not configured")
	}

	if strings.TrimSpace(input.To) == "" {
		return errors.New("email recipient is required")
	}

	if strings.TrimSpace(input.Subject) == "" {
		return errors.New("email subject is required")
	}

	if strings.TrimSpace(input.HTMLBody) == "" {
		return errors.New("email body is required")
	}

	return s.provider.Send(
		ctx,
		strings.TrimSpace(input.To),
		strings.TrimSpace(input.Subject),
		input.HTMLBody,
	)
}
