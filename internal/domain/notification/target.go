package notification

type NotificationTargetType string

const (
	TargetAllUsers      NotificationTargetType = "ALL_USERS"
	TargetSingleUser    NotificationTargetType = "SINGLE_USER"
	TargetSelectedUsers NotificationTargetType = "SELECTED_USERS"
	TargetSegment       NotificationTargetType = "SEGMENT"
)

type NotificationTarget struct {
	Type    NotificationTargetType
	UserIDs []string
}

func (t NotificationTarget) IsValid() bool {
	switch t.Type {
	case TargetAllUsers:
		return len(t.UserIDs) == 0

	case TargetSingleUser:
		return len(t.UserIDs) == 1

	case TargetSelectedUsers:
		return len(t.UserIDs) > 0

	case TargetSegment:
		return true

	default:
		return false
	}
}
