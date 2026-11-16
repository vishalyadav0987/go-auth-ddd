
DROP INDEX IF EXISTS idx_notification_deliveries_processing_token;

ALTER TABLE notification_deliveries
DROP COLUMN IF EXISTS processing_token;
