-- +migrate Down

ALTER TABLE notification_deliveries
DROP CONSTRAINT IF EXISTS chk_notification_deliveries_retry_cycle;

ALTER TABLE notification_deliveries
DROP COLUMN IF EXISTS retry_cycle;