package notification

import (
	"errors"
	"strings"
)

func (n *Notification) Validate() error {
	if n == nil {
		return errors.New("notification is required")
	}

	if !n.Type.IsValid() {
		return ErrInvalidType
	}

	if !n.Target.IsValid() {
		return ErrInvalidTarget
	}

	if !n.Priority.IsValid() {
		return ErrInvalidPriority
	}

	if len(n.Channels) == 0 {
		return ErrNoChannelsSpecified
	}

	seen := make(map[NotificationChannel]struct{})

	for _, channel := range n.Channels {
		if !channel.IsValid() {
			return ErrInvalidChannel
		}

		if _, exists := seen[channel]; exists {
			return errors.New("duplicate notification channel")
		}

		seen[channel] = struct{}{}
	}

	if strings.TrimSpace(n.Title) == "" &&
		containsChannel(n.Channels, ChannelPush) {
		return errors.New("push notification title is required")
	}

	if strings.TrimSpace(n.Body) == "" &&
		containsChannel(n.Channels, ChannelPush) {
		return errors.New("push notification body is required")
	}

	if strings.TrimSpace(n.HTMLBody) == "" &&
		containsChannel(n.Channels, ChannelEmail) {
		return errors.New("email HTML body is required")
	}

	return nil
}

func containsChannel(
	channels []NotificationChannel,
	target NotificationChannel,
) bool {
	for _, channel := range channels {
		if channel == target {
			return true
		}
	}

	return false
}
