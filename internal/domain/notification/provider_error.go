// Currently, our notification system treats provider errors mostly the same way. That is not a good production strategy.

// Consider these examples:
// Error	Correct behavior
// SMTP connection refused	Retry
// SMTP temporary 4xx response	Retry
// Invalid recipient address	Permanent failure
// SMTP authentication rejected	Permanent/configuration failure; alert
// FCM invalid registration token	Permanent for that token; deactivate it
// FCM unavailable	Retry
// Provider timeout	Potentially ambiguous; handle carefully

// We need a centralized error classification mechanism.

package notification

import (
	"errors"
	"time"
)

type ProviderErrorKind string

const (
	ProviderErrorTransient ProviderErrorKind = "TRANSIENT"
	ProviderErrorPermanent ProviderErrorKind = "PERMANENT"
	ProviderErrorAmbiguous ProviderErrorKind = "AMBIGUOUS"
)

type ProviderError struct {
	Kind       ProviderErrorKind
	Code       string
	Message    string
	Err        error
	RetryAfter time.Duration
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}

	return e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

func NewProviderError(
	kind ProviderErrorKind,
	code string,
	message string,
	err error,
) *ProviderError {
	return &ProviderError{
		Kind:    kind,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func IsProviderError(err error) (*ProviderError, bool) {
	var providerErr *ProviderError

	if errors.As(err, &providerErr) {
		return providerErr, true
	}

	return nil, false
}
