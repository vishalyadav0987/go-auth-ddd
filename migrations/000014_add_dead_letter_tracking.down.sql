DROP INDEX IF EXISTS idx_notification_deliveries_dead_lettered;

ALTER TABLE notification_deliveries
DROP COLUMN IF EXISTS dead_lettered_at;