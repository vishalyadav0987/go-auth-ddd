-- +migrate Up

-- Tracks how many manual retry cycles have been started.
-- The initial delivery belongs to retry cycle 0.
ALTER TABLE notification_deliveries
ADD COLUMN retry_cycle INTEGER NOT NULL DEFAULT 0;

-- Prevent negative retry-cycle values.
ALTER TABLE notification_deliveries
ADD CONSTRAINT chk_notification_deliveries_retry_cycle
CHECK (retry_cycle >= 0);