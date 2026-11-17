package notificationQueue

import (
	"context"

	appnotification "github.com/vishalyadav0987/authentication/internal/application/notification"
)

// DeliveryProcessor is the application use case that processes one
// notification delivery. ProcessDeliveryUseCase implements it.
type DeliveryProcessor interface {
	Execute(
		ctx context.Context,
		task appnotification.DeliveryTask,
	) error
}

var _ DeliveryProcessor = (*appnotification.ProcessDeliveryUseCase)(nil)
