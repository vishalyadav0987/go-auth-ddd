
ALTER TABLE notification_deliveries
ADD COLUMN processing_token UUID;

CREATE INDEX idx_notification_deliveries_processing_token
ON notification_deliveries (processing_token)
WHERE processing_token IS NOT NULL;

-- Add a unique processing ownership token
-- Before we update the worker, we need to solve the stale-worker problem.

-- Consider this scenario:
-- - Worker A claims delivery D1.
-- - Worker A becomes unresponsive.
-- - After two minutes, Worker B reclaims D1.
-- - Worker A wakes up and tries to update the delivery.
-- We must prevent Worker A from modifying a delivery now owned by Worker B.

-- We'll introduce a processing token (also called a lease token).