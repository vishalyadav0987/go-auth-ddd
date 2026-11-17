-- =========================================
-- 1. User inbox
-- =========================================
-- One row per (user, notification). Read state belongs to the user, not
-- to a channel delivery: a notification sent by email AND push is still
-- a single inbox item.
CREATE TABLE IF NOT EXISTS user_notifications (
    user_id TEXT NOT NULL,
    notification_id TEXT NOT NULL,

    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, notification_id),

    CONSTRAINT fk_user_notifications_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT fk_user_notifications_notification
        FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE
);

-- Inbox listing: newest first, keyset-paginated.
CREATE INDEX IF NOT EXISTS idx_user_notifications_inbox
    ON user_notifications (user_id, created_at DESC, notification_id DESC);

-- Unread badge count.
CREATE INDEX IF NOT EXISTS idx_user_notifications_unread
    ON user_notifications (user_id)
    WHERE read_at IS NULL;


-- =========================================
-- 2. Notification preferences
-- =========================================
-- A missing row means "everything enabled". Mandatory types (OTP,
-- SECURITY_ALERT) are enforced in the application and ignore these flags.
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,

    email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    push_enabled BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, type),

    CONSTRAINT fk_notification_preferences_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT chk_notification_preferences_type
        CHECK (type IN (
            'OTP',
            'ORDER_CREATED',
            'PAYMENT_SUCCESS',
            'PAYMENT_FAILED',
            'INVESTMENT_UPDATE',
            'SECURITY_ALERT',
            'ANNOUNCEMENT',
            'MARKETING'
        ))
);


-- =========================================
-- 3. User devices (FCM tokens)
-- =========================================
-- An FCM token identifies one app install, so it is globally unique.
-- Registering a token already owned by another user moves it to the new
-- user (same phone, different account).
CREATE TABLE IF NOT EXISTS user_devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,

    fcm_token TEXT NOT NULL,
    platform VARCHAR(20) NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_devices_fcm_token UNIQUE (fcm_token),

    CONSTRAINT fk_user_devices_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT chk_user_devices_platform
        CHECK (platform IN ('ANDROID', 'IOS'))
);

CREATE INDEX IF NOT EXISTS idx_user_devices_active_user
    ON user_devices (user_id)
    WHERE is_active = TRUE;
