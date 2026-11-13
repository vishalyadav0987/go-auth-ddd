package notification

import "context"

// EmailProvider defines the contract for sending emails.
type EmailProvider interface {
	Send(
		ctx context.Context,
		to string,
		subject string,
		htmlBody string,
	) error
}
