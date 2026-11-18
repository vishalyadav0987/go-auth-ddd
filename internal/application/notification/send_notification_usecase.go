package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
)

// ErrInvalidSendRequest wraps notification validation failures so callers
// (e.g. HTTP handlers) can report them as client errors.
var ErrInvalidSendRequest = errors.New("invalid notification")

// RecipientResolver returns the users eligible for a notification on one
// channel. ResolveRecipientsUseCase implements it.
type RecipientResolver interface {
	Execute(
		ctx context.Context,
		target domain.NotificationTarget,
		notificationType domain.NotificationType,
		channel domain.NotificationChannel,
	) ([]string, error)
}

var _ RecipientResolver = (*ResolveRecipientsUseCase)(nil)

// EmailDirectory looks up users' email addresses in one query.
type EmailDirectory interface {
	GetEmailsByUserIDs(ctx context.Context, userIDs []string) (map[string]string, error)
}

// PushTokenDirectory looks up users' active FCM tokens in one query.
type PushTokenDirectory interface {
	GetActiveTokensByUserIDs(ctx context.Context, userIDs []string) (map[string][]string, error)
}

// SendNotificationResult summarises what was created.
type SendNotificationResult struct {
	NotificationID  string
	RecipientCount  int
	EmailDeliveries int
	PushDeliveries  int
	ScheduledAt     *time.Time
}

// SendNotificationUseCase creates a notification and schedules its
// delivery. It does not send anything itself:
//
//  1. Validate the notification.
//  2. For each channel, resolve eligible users (preferences + reachability).
//  3. Build deliveries: one per user for EMAIL, one per active device for PUSH.
//  4. Atomically store the notification, deliveries, inbox entries and a
//     NOTIFICATION_CREATED outbox event.
//
// The notification outbox worker then publishes the deliveries to Asynq.
type SendNotificationUseCase struct {
	notificationRepository domain.NotificationRepository
	recipientResolver      RecipientResolver
	emails                 EmailDirectory
	pushTokens             PushTokenDirectory
}

func NewSendNotificationUseCase(
	notificationRepository domain.NotificationRepository,
	recipientResolver RecipientResolver,
	emails EmailDirectory,
	pushTokens PushTokenDirectory,
) *SendNotificationUseCase {
	return &SendNotificationUseCase{
		notificationRepository: notificationRepository,
		recipientResolver:      recipientResolver,
		emails:                 emails,
		pushTokens:             pushTokens,
	}
}

// Execute creates the notification on every channel it lists.
//
// A channel with no eligible recipients is skipped; domain.ErrNoRecipients
// is returned only if no channel produced a delivery. If ScheduledAt is in
// the future, delivery publishing is delayed until then.
func (uc *SendNotificationUseCase) Execute(
	ctx context.Context,
	n *domain.Notification,
) (*SendNotificationResult, error) {

	if uc.notificationRepository == nil || uc.recipientResolver == nil ||
		uc.emails == nil || uc.pushTokens == nil {
		return nil, errors.New("send notification use case is not configured")
	}

	if n == nil {
		return nil, fmt.Errorf("%w: notification is required", ErrInvalidSendRequest)
	}

	now := time.Now().UTC()

	if n.ID == "" {
		n.ID = uuid.NewString()
	}

	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}
	n.UpdatedAt = now

	start := time.Now()

	ctx = notiflog.With(ctx, notiflog.KeyNotificationID, n.ID)
	lg := notiflog.From(ctx).With(
		"notification_type", n.Type,
		"priority", n.Priority,
		"target_type", n.Target.Type,
		"target_user_count", len(n.Target.UserIDs),
		"channels", n.Channels,
	)

	lg.Info("send.started", "scheduled_at", n.ScheduledAt)

	if err := n.Validate(); err != nil {
		lg.Warn("send.validation_failed", notiflog.Err(err))
		return nil, fmt.Errorf("%w: %v", ErrInvalidSendRequest, err)
	}

	result := &SendNotificationResult{NotificationID: n.ID}
	recipients := make(map[string]struct{})

	var deliveries []*domain.NotificationDelivery

	newDelivery := func(userID string, channel domain.NotificationChannel, recipient string) {
		deliveries = append(deliveries, &domain.NotificationDelivery{
			ID:             uuid.NewString(),
			NotificationID: n.ID,
			UserID:         userID,
			Channel:        channel,
			Status:         domain.DeliveryPending,
			Recipient:      recipient,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
		recipients[userID] = struct{}{}
	}

	for _, channel := range n.Channels {
		userIDs, err := uc.recipientResolver.Execute(ctx, n.Target, n.Type, channel)
		if errors.Is(err, domain.ErrNoRecipients) {
			lg.Info("send.channel_skipped",
				notiflog.KeyChannel, channel,
				"reason", "no eligible users (preferences / no email / no active device)",
			)
			continue
		}
		if err != nil {
			lg.Error("send.recipient_resolution_failed", notiflog.KeyChannel, channel, notiflog.Err(err))
			return nil, fmt.Errorf("resolve %s recipients: %w", channel, err)
		}

		before := len(deliveries)

		switch channel {
		case domain.ChannelEmail:
			emails, err := uc.emails.GetEmailsByUserIDs(ctx, userIDs)
			if err != nil {
				lg.Error("send.email_lookup_failed", notiflog.Err(err))
				return nil, fmt.Errorf("look up recipient emails: %w", err)
			}

			for _, userID := range userIDs {
				if email, ok := emails[userID]; ok && email != "" {
					newDelivery(userID, channel, email)
					result.EmailDeliveries++
				}
			}

		case domain.ChannelPush:
			tokens, err := uc.pushTokens.GetActiveTokensByUserIDs(ctx, userIDs)
			if err != nil {
				lg.Error("send.device_lookup_failed", notiflog.Err(err))
				return nil, fmt.Errorf("look up recipient devices: %w", err)
			}

			for _, userID := range userIDs {
				for _, token := range tokens[userID] {
					newDelivery(userID, channel, token)
					result.PushDeliveries++
				}
			}

		default:
			return nil, fmt.Errorf("%w: unsupported channel %s", ErrInvalidSendRequest, channel)
		}

		lg.Info("send.channel_resolved",
			notiflog.KeyChannel, channel,
			"eligible_users", len(userIDs),
			"deliveries", len(deliveries)-before,
		)
	}

	if len(deliveries) == 0 {
		lg.Warn("send.no_recipients", notiflog.Since(start))
		return nil, domain.ErrNoRecipients
	}

	result.RecipientCount = len(recipients)

	payload, err := json.Marshal(domain.NotificationCreatedEvent{NotificationID: n.ID})
	if err != nil {
		return nil, fmt.Errorf("marshal notification created event: %w", err)
	}

	event := &domain.NotificationOutboxEvent{
		ID:          uuid.NewString(),
		EventType:   domain.OutboxEventNotificationCreated,
		AggregateID: n.ID,
		Payload:     payload,
		CreatedAt:   now,
	}

	// A scheduled notification's event only becomes due at its scheduled
	// time, so its deliveries are published then, not now.
	if n.ScheduledAt != nil && n.ScheduledAt.After(now) {
		scheduledAt := n.ScheduledAt.UTC()
		event.NextRetryAt = &scheduledAt
		result.ScheduledAt = &scheduledAt
	}

	if err := uc.notificationRepository.CreateNotificationWithDeliveriesAndOutbox(
		ctx, n, deliveries, event,
	); err != nil {
		lg.Error("send.store_failed", notiflog.Err(err))
		return nil, fmt.Errorf("store notification: %w", err)
	}

	lg.Info("send.stored",
		"recipients", result.RecipientCount,
		"email_deliveries", result.EmailDeliveries,
		"push_deliveries", result.PushDeliveries,
		notiflog.KeyOutboxEventID, event.ID,
		"publish_at", event.NextRetryAt,
		notiflog.Since(start),
	)

	// Per-delivery IDs to trace every single delivery.
	for _, d := range deliveries {
		lg.Info("send.delivery_created",
			notiflog.KeyDeliveryID, d.ID,
			notiflog.KeyUserID, d.UserID,
			notiflog.KeyChannel, d.Channel,
			notiflog.KeyRecipient, notiflog.MaskRecipient(string(d.Channel), d.Recipient),
		)
	}

	return result, nil
}
