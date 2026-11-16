
DROP INDEX IF EXISTS idx_notification_deliveries_processing_started;

ALTER TABLE notification_deliveries
DROP COLUMN IF EXISTS processing_started_at;
