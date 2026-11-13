package notification

import "context"

type NotificationTemplateRepository interface {
	GetActiveTemplate(
		ctx context.Context,
		code string,
		channel NotificationChannel,
	) (*NotificationTemplate, error)

	CreateTemplate(
		ctx context.Context,
		template *NotificationTemplate,
	) error

	UpdateTemplate(
		ctx context.Context,
		template *NotificationTemplate,
	) error
}
