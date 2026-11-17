-- Dedicated transactional outbox for the notification module.
--
-- Order/payment events stay in outbox_events and are consumed by the
-- order outbox worker. Notification events (e.g. DELIVERY_RETRY_REQUESTED)
-- live here so the two workers never claim each other's events.

CREATE TABLE IF NOT EXISTS notification_outbox_events (
    id TEXT PRIMARY KEY,

    event_type VARCHAR(100) NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload JSONB NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,

    -- Processing lease: when the event was claimed and by whom.
    processing_started_at TIMESTAMPTZ,
    processing_token UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT chk_notification_outbox_status
        CHECK (status IN ('PENDING', 'PROCESSING', 'PROCESSED', 'FAILED')),

    CONSTRAINT chk_notification_outbox_attempts
        CHECK (attempts >= 0),

    -- Only PROCESSING events may hold a lease.
    CONSTRAINT chk_notification_outbox_processing_token
        CHECK (
            (status = 'PROCESSING' AND processing_token IS NOT NULL
                AND processing_started_at IS NOT NULL)
            OR
            (status <> 'PROCESSING' AND processing_token IS NULL)
        )
);

-- Pending events that are due for (re)processing.
CREATE INDEX IF NOT EXISTS idx_notification_outbox_due
    ON notification_outbox_events (next_retry_at, created_at)
    WHERE status = 'PENDING';

-- PROCESSING events whose lease may have expired.
CREATE INDEX IF NOT EXISTS idx_notification_outbox_processing_lease
    ON notification_outbox_events (processing_started_at)
    WHERE status = 'PROCESSING';

-- Terminally failed events, for operator inspection.
CREATE INDEX IF NOT EXISTS idx_notification_outbox_failed
    ON notification_outbox_events (updated_at DESC)
    WHERE status = 'FAILED';
