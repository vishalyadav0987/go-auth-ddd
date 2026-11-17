-- Each worker claim gets a unique ownership token.
ALTER TABLE outbox_events
ADD COLUMN processing_token UUID NULL;

-- Only PROCESSING events should have an active token.
ALTER TABLE outbox_events
ADD CONSTRAINT chk_outbox_processing_token
CHECK (
    (status = 'PROCESSING' AND processing_token IS NOT NULL)
    OR
    (status <> 'PROCESSING' AND processing_token IS NULL)
);