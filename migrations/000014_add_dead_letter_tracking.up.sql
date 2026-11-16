ALTER TABLE notification_deliveries
ADD COLUMN dead_lettered_at TIMESTAMPTZ NULL;

CREATE INDEX idx_notification_deliveries_dead_lettered
ON notification_deliveries (dead_lettered_at DESC)
WHERE dead_lettered_at IS NOT NULL;