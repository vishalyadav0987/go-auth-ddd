
ALTER TABLE notification_deliveries
DROP CONSTRAINT IF EXISTS chk_delivery_status;

ALTER TABLE notification_deliveries
ADD CONSTRAINT chk_delivery_status
CHECK (
    status IN (
        'PENDING',
        'PROCESSING',
        'RETRYING',
        'ACCEPTED',
        'DELIVERED',
        'FAILED',
        'CANCELLED',
        'UNKNOWN'
    )
);
