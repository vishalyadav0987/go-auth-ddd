
CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,

    payment_id VARCHAR(100) NOT NULL UNIQUE,
    order_id VARCHAR(100) NOT NULL UNIQUE,
    user_id VARCHAR(100) NOT NULL,

    cashfree_order_id VARCHAR(255) NOT NULL UNIQUE,
    cashfree_payment_id VARCHAR(255),
    payment_session_id TEXT NOT NULL,

    amount_paise BIGINT NOT NULL CHECK (amount_paise > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',

    status VARCHAR(30) NOT NULL DEFAULT 'CREATED',

    idempotency_key VARCHAR(255) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_payments_order
        FOREIGN KEY (order_id)
        REFERENCES orders(order_id),

    CONSTRAINT chk_payments_currency
        CHECK (currency = 'INR'),

    CONSTRAINT chk_payments_status
        CHECK (
            status IN (
                'CREATED',
                'PENDING',
                'SUCCESS',
                'FAILED',
                'CANCELLED'
            )
        ),

    CONSTRAINT uq_payments_user_idempotency
        UNIQUE (user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id
    ON payments(order_id);

CREATE INDEX IF NOT EXISTS idx_payments_user_id
    ON payments(user_id);

CREATE INDEX IF NOT EXISTS idx_payments_status
    ON payments(status);

CREATE INDEX IF NOT EXISTS idx_payments_cashfree_order_id
    ON payments(cashfree_order_id);

CREATE INDEX IF NOT EXISTS idx_payments_cashfree_payment_id
    ON payments(cashfree_payment_id);
