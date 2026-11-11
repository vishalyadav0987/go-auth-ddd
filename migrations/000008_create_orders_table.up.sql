
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(100) NOT NULL UNIQUE,
    user_id VARCHAR(100) NOT NULL,
    scheme_code VARCHAR(100) NOT NULL,
    amount_paise BIGINT NOT NULL CHECK (amount_paise > 0),
    investment_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    idempotency_key VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_orders_user_idempotency
        UNIQUE (user_id, idempotency_key),

    CONSTRAINT chk_orders_investment_type
        CHECK (investment_type IN ('LUMPSUM', 'SIP')),

    CONSTRAINT chk_orders_status
        CHECK (
            status IN (
                'PENDING',
                'PROCESSING',
                'COMPLETED',
                'FAILED',
                'CANCELLED'
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id
    ON orders(user_id);

CREATE INDEX IF NOT EXISTS idx_orders_status
    ON orders(status);
