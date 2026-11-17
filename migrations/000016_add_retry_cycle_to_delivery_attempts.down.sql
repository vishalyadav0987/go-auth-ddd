DROP INDEX IF EXISTS idx_delivery_attempts_delivery_cycle;

ALTER TABLE notification_delivery_attempts
DROP CONSTRAINT IF EXISTS chk_delivery_attempts_retry_cycle;

ALTER TABLE notification_delivery_attempts
DROP COLUMN IF EXISTS retry_cycle;