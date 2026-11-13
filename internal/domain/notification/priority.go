package notification

type NotificationPriority string

const (
	PriorityHigh   NotificationPriority = "HIGH"
	PriorityNormal NotificationPriority = "NORMAL"
	PriorityLow    NotificationPriority = "LOW"
)

func (p NotificationPriority) QueueName() string {
	switch p {
	case PriorityHigh:
		return "critical"
	case PriorityNormal:
		return "default"
	case PriorityLow:
		return "low"
	default:
		return ""
	}
}

func (p NotificationPriority) IsValid() bool {
	switch p {
	case PriorityHigh, PriorityNormal, PriorityLow:
		return true
	default:
		return false
	}
}
