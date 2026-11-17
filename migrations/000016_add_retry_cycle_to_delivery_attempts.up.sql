-- Now we need to associate every attempt with the retry cycle in which it occurred.

-- This allows us to answer two different questions:
-- - How many attempts have ever been made for this delivery?
-- - How many attempts have been made in the current retry cycle?
-- We'll preserve the global attempt number and add a retry_cycle field to every attempt.


-- +migrate Up

ALTER TABLE notification_delivery_attempts
ADD COLUMN retry_cycle INTEGER NOT NULL DEFAULT 0;

ALTER TABLE notification_delivery_attempts
ADD CONSTRAINT chk_delivery_attempts_retry_cycle
CHECK (retry_cycle >= 0);

CREATE INDEX idx_delivery_attempts_delivery_cycle
ON notification_delivery_attempts (
    delivery_id,
    retry_cycle,
    attempt_number
);