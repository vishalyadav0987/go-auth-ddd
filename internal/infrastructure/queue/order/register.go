package orderQueue

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// RegisterHandlers registers the order event handlers on the shared
// Asynq ServeMux.
func RegisterHandlers(mux *asynq.ServeMux) error {
	if mux == nil {
		return fmt.Errorf("asynq serve mux is required")
	}

	mux.HandleFunc(
		TaskTypeOrderEvent,
		NewOrderEventHandler().HandleOrderEvent,
	)

	return nil
}
