
-- Add a timestamp to track when an outbox event was claimed.
ALTER TABLE outbox_events
ADD COLUMN processing_started_at TIMESTAMPTZ NULL;

-- Helps workers find events whose processing lease has expired.
CREATE INDEX idx_outbox_events_processing_lease
ON outbox_events (processing_started_at)
WHERE status = 'PROCESSING';