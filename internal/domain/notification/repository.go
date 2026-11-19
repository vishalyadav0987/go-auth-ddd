package notification

import "context"

type NotificationRepository interface {
	CreateNotification(
		ctx context.Context,
		notification *Notification,
	) error

	GetNotificationByID(
		ctx context.Context,
		id string,
	) (*Notification, error)

	CreateDeliveries(
		ctx context.Context,
		deliveries []*NotificationDelivery,
	) error

	GetDeliveryByID(
		ctx context.Context,
		id string,
	) (*NotificationDelivery, error)

	// Add the processingToken parameter to the UpdateDeliveryStatus method. This ensures that only the worker that has claimed the delivery can update its status, preventing race conditions and ensuring data integrity.
	// Problem: Worker A claims a delivery. Its lease expires, and Worker B reclaims it. If Worker A later executes UpdateDeliveryStatus, it could overwrite Worker B's result.

	UpdateDeliveryStatus(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		status DeliveryStatus,
	) error

	UpdateDeliveryProviderID(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		providerID string,
	) error

	IncrementDeliveryAttempt(
		ctx context.Context,
		deliveryID string,
	) error

	GetPendingDeliveries(
		ctx context.Context,
		limit int,
	) ([]*NotificationDelivery, error)

	UpdateDeliveryRetry(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		status DeliveryStatus,
		nextRetryAt int64,
		reason string,
	) error

	// 	Before we build the retry worker, we need to fix an important production issue: two workers could process the same notification delivery simultaneously.

	// Currently, the flow is:
	// 1. Worker A loads a delivery and sees PENDING.
	// 2. Worker B loads the same delivery and also sees PENDING.
	// 3. Both workers send the email or push notification.
	// This can cause duplicate messages.

	// We'll solve this using an atomic PostgreSQL update.

	// 	Result	Meaning
	// true, nil	This worker successfully claimed the delivery
	// false, nil	Another worker already claimed it, or it isn't eligible
	// false, err	Database operation failed

	//
	// expectedRetryCycle makes the claim fail (false, nil) when the
	// delivery has moved to another retry cycle since the task was queued.
	ClaimDelivery(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		expectedRetryCycle int,
	) (bool, error)

	// 	Before implementing the retry worker, we need to fix a concurrency and consistency issue in our current processing pipeline.

	// Currently, we execute two separate database operations:
	// 1. IncrementDeliveryAttempt()
	// 2. CreateAttempt()
	// If the first succeeds but the second fails, PostgreSQL will show an increased attempt count without a corresponding attempt-history record.

	// We should combine these operations into one transaction.

	// After Procession gtoken Currently, our CreateDeliveryAttempt() method checks whether the delivery is PROCESSING, but that is not enough. An expired worker could still create an attempt after another worker has reclaimed the delivery.

	CreateDeliveryAttempt(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		attempt *NotificationDeliveryAttempt,
	) (int, error)

	// 	So far, our delivery lease expires after 2 minutes:
	// processing_started_at < NOW() - INTERVAL '2 minutes'

	// This creates a problem.

	// Imagine an email provider takes 2 minutes and 30 seconds to respond. Worker A is still working, but Worker B considers its lease expired and reclaims the delivery.

	// Both workers may now send the same notification.

	// We need a better lease mechanism.
	// 1. Understand the heartbeat concept
	// A heartbeat is a periodic database update that tells the system:
	// This worker is still alive and actively processing this delivery.

	// Example: 5-minute lease
	// 1. 00:00 — Claim

	//    Worker A claims delivery with token A.
	//    Lease expires at 05:00
	// 2. 01:00 — Heartbeat

	//    Worker A updates processing_started_at.
	//    Lease expires at 06:00
	// 3. 02:00 — Heartbeat

	//    Worker A is still processing.
	//    Lease expires at 07:00
	// 4. 03:00 — Provider responds

	//    Worker A saves the result and completes the delivery.
	//    Lease released

	// HeartbeatDelivery renews the lease only while the caller still owns
	// the delivery (token) and it is still in the expected retry cycle.
	HeartbeatDelivery(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		expectedRetryCycle int,
	) (bool, error)

	// 	In Step 99, we added dead_lettered_at. Now we need to ensure that when a delivery exhausts its retry attempts, both fields are updated together:
	// - status = FAILED
	// - dead_lettered_at = NOW()
	// We should do this through the repository, not by directly writing SQL inside the use case.
	MarkDeliveryDeadLettered(
		ctx context.Context,
		deliveryID string,
		processingToken string,
		reason string,
	) error

	// GetDeadLetterDeliveries retrieves dead-lettered notification
	// deliveries using pagination.
	//
	// limit  = maximum number of records to return.
	// offset = number of records to skip.
	GetDeadLetterDeliveries(
		ctx context.Context,
		limit int,
		offset int,
	) ([]*DeadLetterDelivery, error)

	// PrepareDeadLetterRetry atomically moves a dead-lettered delivery
	// back to PENDING so it can be scheduled for another delivery cycle.
	//
	// It must preserve the existing delivery attempt history.
	// It must reject deliveries that are not currently in the DLQ.
	//
	// adminID is recorded on the retry outbox event. The returned int is
	// the previous retry cycle; the delivery carries the new one.
	PrepareDeadLetterRetry(
		ctx context.Context,
		deliveryID string,
		adminID string,
	) (*NotificationDelivery, int, error)

	// CreateNotificationWithDeliveriesAndOutbox atomically creates a
	// notification, its delivery records, and its outbox event.
	//
	// All three operations must succeed together. If any operation fails,
	// the repository must roll back the entire transaction.

	CreateNotificationWithDeliveriesAndOutbox(
		ctx context.Context,
		notification *Notification,
		deliveries []*NotificationDelivery,
		event *NotificationOutboxEvent,
	) error

	// GetPendingDeliveriesByNotificationID returns the notification's
	// deliveries that are still PENDING.
	GetPendingDeliveriesByNotificationID(
		ctx context.Context,
		notificationID string,
	) ([]*NotificationDelivery, error)

	// 	We need a PostgreSQL operation that performs these two actions atomically:
	// - Finds due RETRYING deliveries.
	// - Changes them to PENDING and inserts their automatic retry outbox events.
	// This prevents multiple scheduler instances from repeatedly scheduling the same delivery.

	ScheduleDueDeliveryRetries(
		ctx context.Context,
		limit int,
	) (int, error)
}
