package notificationQueue

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// RegisterHandlers registers the notification task handlers on the shared
// Asynq ServeMux. Email and push share one handler: the stored delivery
// record decides which provider is used.
func RegisterHandlers(
	mux *asynq.ServeMux,
	processor DeliveryProcessor,
) error {

	if mux == nil {
		return fmt.Errorf("asynq serve mux is required")
	}

	if processor == nil {
		return fmt.Errorf("delivery processor is required")
	}

	handler := NewDeliveryHandler(processor)

	mux.HandleFunc(TaskTypeSendEmail, handler.Handle)
	mux.HandleFunc(TaskTypeSendPush, handler.Handle)

	return nil
}
