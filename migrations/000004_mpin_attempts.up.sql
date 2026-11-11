CREATE TABLE IF NOT EXISTS mpin_attempts (
    user_id TEXT,
    otp_access_token TEXT,
    attempts INTEGER DEFAULT 0,
    blocked_until TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (user_id, otp_access_token)
);