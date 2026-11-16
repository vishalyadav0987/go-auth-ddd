package notification

import (
	"context"
	"errors"
	"net"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

type ProviderFailureDecision string

const (
	FailureRetry     ProviderFailureDecision = "RETRY"
	FailurePermanent ProviderFailureDecision = "PERMANENT"
	FailureAmbiguous ProviderFailureDecision = "AMBIGUOUS"
)

func domainProviderError(err error) (*domain.ProviderError, bool) {
	return domain.IsProviderError(err)
}

func ClassifyProviderError(err error) ProviderFailureDecision {
	if err == nil {
		return ""
	}

	providerErr, ok := domainProviderError(err)
	if ok {
		switch providerErr.Kind {
		case domain.ProviderErrorTransient:
			return FailureRetry

		case domain.ProviderErrorPermanent:
			return FailurePermanent

		case domain.ProviderErrorAmbiguous:
			return FailureAmbiguous
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return FailureAmbiguous
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return FailureAmbiguous
		}

		return FailureRetry
	}

	return FailureAmbiguous
}

// ProcessDelivery
//       |
//       v
// Email / Push Provider
//       |
//       v
// Provider Adapter
//       |
//       v
// ProviderError
//       |
//       v
// ClassifyProviderError
//       |
//       +---- TRANSIENT  -> Retry
//       |
//       +---- PERMANENT  -> Fail
//       |
//       +---- AMBIGUOUS  -> Reconciliation
