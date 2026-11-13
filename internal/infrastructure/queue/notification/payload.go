package notificationQueue

import (
	"encoding/json"
	"fmt"
)

const (
	TaskTypeSendEmail = "notification:send_email"
	TaskTypeSendPush  = "notification:send_push"
)

type DeliveryTaskPayload struct {
	DeliveryID     string `json:"delivery_id"`
	NotificationID string `json:"notification_id"`
	UserID         string `json:"user_id"`
	Channel        string `json:"channel"`
}

func MarshalDeliveryPayload(
	payload DeliveryTaskPayload,
) ([]byte, error) {
	if payload.DeliveryID == "" ||
		payload.NotificationID == "" ||
		payload.UserID == "" ||
		payload.Channel == "" {
		return nil, fmt.Errorf("invalid notification delivery payload")
	}

	return json.Marshal(payload)
}

func UnmarshalDeliveryPayload(
	data []byte,
) (DeliveryTaskPayload, error) {
	var payload DeliveryTaskPayload

	if err := json.Unmarshal(data, &payload); err != nil {
		return DeliveryTaskPayload{}, fmt.Errorf(
			"decode notification delivery payload: %w",
			err,
		)
	}

	if payload.DeliveryID == "" ||
		payload.NotificationID == "" ||
		payload.UserID == "" ||
		payload.Channel == "" {
		return DeliveryTaskPayload{}, fmt.Errorf(
			"invalid notification delivery payload",
		)
	}

	return payload, nil
}
