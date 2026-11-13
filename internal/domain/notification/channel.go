package notification

type NotificationChannel string

const (
	ChannelEmail NotificationChannel = "EMAIL"
	ChannelPush  NotificationChannel = "PUSH"
)

func (c NotificationChannel) IsValid() bool {
	switch c {
	case ChannelEmail, ChannelPush:
		return true
	default:
		return false
	}
}
