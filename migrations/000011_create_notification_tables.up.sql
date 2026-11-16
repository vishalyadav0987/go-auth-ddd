
-- =========================================
-- 1. Notifications
-- =========================================

CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,

    type VARCHAR(50) NOT NULL,
    title TEXT,
    body TEXT,
    email_subject TEXT,
    html_body TEXT,

    target_type VARCHAR(30) NOT NULL,
    target_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb,

    channels JSONB NOT NULL,
    priority VARCHAR(20) NOT NULL,

    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    scheduled_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_notifications_type
        CHECK (type IN (
            'OTP',
            'ORDER_CREATED',
            'PAYMENT_SUCCESS',
            'PAYMENT_FAILED',
            'INVESTMENT_UPDATE',
            'SECURITY_ALERT',
            'ANNOUNCEMENT',
            'MARKETING'
        )),

    CONSTRAINT chk_notifications_target_type
        CHECK (target_type IN (
            'ALL_USERS',
            'SINGLE_USER',
            'SELECTED_USERS',
            'SEGMENT'
        )),

    CONSTRAINT chk_notifications_priority
        CHECK (priority IN ('HIGH', 'NORMAL', 'LOW')),

    CONSTRAINT chk_notifications_channels
        CHECK (
            jsonb_typeof(channels) = 'array'
            AND jsonb_array_length(channels) > 0
        ),

    CONSTRAINT chk_notifications_target_user_ids
        CHECK (jsonb_typeof(target_user_ids) = 'array')
);

CREATE INDEX IF NOT EXISTS idx_notifications_created_at
    ON notifications (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_scheduled_at
    ON notifications (scheduled_at)
    WHERE scheduled_at IS NOT NULL;


-- =========================================
-- 2. Notification Deliveries
-- =========================================

CREATE TABLE IF NOT EXISTS notification_deliveries (
    id TEXT PRIMARY KEY,

    notification_id TEXT NOT NULL,
    user_id TEXT NOT NULL,

    channel VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',

    recipient TEXT NOT NULL,

    provider_id TEXT,
    failure_reason TEXT,

    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ,

    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_delivery_notification
        FOREIGN KEY (notification_id)
        REFERENCES notifications(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_delivery_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_delivery_channel
        CHECK (channel IN ('EMAIL', 'PUSH')),

    CONSTRAINT chk_delivery_status
        CHECK (status IN (
            'PENDING',
            'PROCESSING',
            'RETRYING',
            'ACCEPTED',
            'DELIVERED',
            'FAILED',
            'CANCELLED',
            'UNKNOWN'
        )),

    CONSTRAINT chk_delivery_attempt_count
        CHECK (attempt_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_notification
    ON notification_deliveries (notification_id);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user
    ON notification_deliveries (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_pending
    ON notification_deliveries (status, next_retry_at)
    WHERE status IN ('PENDING', 'RETRYING');


-- =========================================
-- 3. Notification Delivery Attempts
-- =========================================

CREATE TABLE IF NOT EXISTS notification_delivery_attempts (
    id TEXT PRIMARY KEY,

    delivery_id TEXT NOT NULL,

    attempt_number INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,

    provider_id TEXT,
    error_message TEXT,

    attempted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_delivery_attempt
        FOREIGN KEY (delivery_id)
        REFERENCES notification_deliveries(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_attempt_number
        CHECK (attempt_number > 0),

    CONSTRAINT chk_attempt_status
        CHECK (status IN (
            'PROCESSING',
            'ACCEPTED',
            'FAILED'
        )),

    CONSTRAINT uq_delivery_attempt_number
        UNIQUE (delivery_id, attempt_number)
);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_delivery
    ON notification_delivery_attempts (
        delivery_id,
        attempt_number DESC
    );
