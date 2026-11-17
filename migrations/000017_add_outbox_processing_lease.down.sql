
DROP INDEX IF EXISTS idx_outbox_events_processing_lease;
ALTER TABLE outbox_events DROP COLUMN processing_started_at;