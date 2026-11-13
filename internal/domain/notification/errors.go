package notification

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrDeliveryNotFound     = errors.New("notification delivery not found")

	ErrInvalidNotification = errors.New("invalid notification")
	ErrInvalidTarget       = errors.New("invalid notification target")
	ErrInvalidChannel      = errors.New("invalid notification channel")
	ErrInvalidPriority     = errors.New("invalid notification priority")
	ErrInvalidType         = errors.New("invalid notification type")

	ErrNoChannelsSpecified = errors.New("at least one notification channel is required")
	ErrNoRecipients        = errors.New("no notification recipients found")

	ErrDeliveryNotPending = errors.New("notification delivery is not pending")
	ErrMaxRetriesExceeded = errors.New("maximum delivery retries exceeded")
)
