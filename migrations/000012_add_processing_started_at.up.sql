
ALTER TABLE notification_deliveries
ADD COLUMN processing_started_at TIMESTAMPTZ;

CREATE INDEX idx_notification_deliveries_processing_started
ON notification_deliveries (processing_started_at)
WHERE status = 'PROCESSING';


-- We have fixed concurrent claiming, but there is still an important production failure scenario.

-- Imagine:
-- 1. Worker claims a delivery.
-- 2. PostgreSQL changes its status to PROCESSING.
-- 3. The worker crashes before sending the notification.
-- 4. The delivery remains PROCESSING forever.
-- Our current ClaimDelivery() won't allow another worker to claim it.

-- We need a processing lease: if a worker has been processing a delivery for too long, another worker can reclaim it.

-- First, we'll add a timestamp to track when processing started.

-- Why use a timestamp instead of just changing the status?

-- Because PROCESSING alone cannot tell us whether a worker is actively working or has crashed.

-- Our intended behavior:
-- Processing age	Behavior
-- Under 2 minutes	Leave the delivery alone
-- Over 2 minutes	Allow another worker to reclaim it
-- Provider accepted message but worker crashed	Requires special reconciliation to avoid duplicate sending

-- Important
--  A two-minute lease is only a starting point. We must eventually account for provider timeouts and long-running sends. Reclaiming a delivery after a crash can cause duplicate messages if the provider accepted the previous request but the worker didn't save the result.