
CREATE TABLE IF NOT EXISTS outbox_events (
    id VARCHAR(150) PRIMARY KEY,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT chk_outbox_status
        CHECK (status IN ('PENDING', 'PROCESSING', 'PROCESSED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending_events
    ON outbox_events(status, created_at)
    WHERE status = 'PENDING';
