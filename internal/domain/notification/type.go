package notification

type NotificationType string

const (
	NotificationOTP              NotificationType = "OTP"
	NotificationOrderCreated     NotificationType = "ORDER_CREATED"
	NotificationPaymentSuccess   NotificationType = "PAYMENT_SUCCESS"
	NotificationPaymentFailed    NotificationType = "PAYMENT_FAILED"
	NotificationInvestmentUpdate NotificationType = "INVESTMENT_UPDATE"
	NotificationSecurityAlert    NotificationType = "SECURITY_ALERT"
	NotificationAnnouncement     NotificationType = "ANNOUNCEMENT"
	NotificationMarketing        NotificationType = "MARKETING"
)

func (t NotificationType) IsValid() bool {
	switch t {
	case NotificationOTP,
		NotificationOrderCreated,
		NotificationPaymentSuccess,
		NotificationPaymentFailed,
		NotificationInvestmentUpdate,
		NotificationSecurityAlert,
		NotificationAnnouncement,
		NotificationMarketing:
		return true
	default:
		return false
	}
}
