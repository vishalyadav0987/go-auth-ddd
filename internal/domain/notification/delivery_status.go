package notification

type DeliveryStatus string

const (
	DeliveryPending    DeliveryStatus = "PENDING"
	DeliveryProcessing DeliveryStatus = "PROCESSING"
	DeliveryRetrying   DeliveryStatus = "RETRYING"
	DeliveryAccepted   DeliveryStatus = "ACCEPTED"
	DeliveryDelivered  DeliveryStatus = "DELIVERED"
	DeliveryFailed     DeliveryStatus = "FAILED"
	DeliveryCancelled  DeliveryStatus = "CANCELLED"
)

func (s DeliveryStatus) IsValid() bool {
	switch s {
	case DeliveryPending,
		DeliveryProcessing,
		DeliveryRetrying,
		DeliveryAccepted,
		DeliveryDelivered,
		DeliveryFailed,
		DeliveryCancelled:
		return true
	default:
		return false
	}
}
