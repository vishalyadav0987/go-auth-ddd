-- Down migration:
ALTER TABLE outbox_events DROP CONSTRAINT chk_outbox_processing_token;
ALTER TABLE outbox_events DROP COLUMN processing_token;