// Our PostgreSQL repository is ready. Now we'll create the application use case that controls how the admin retrieves DLQ records.

// The HTTP handler should not directly decide database pagination rules. That responsibility belongs in the application layer.

package notification

import (
	"context"
	"fmt"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

// Default page size for DLQ listing.
const defaultDeadLetterPageSize = 20

// Maximum number of DLQ records allowed per request.
const maxDeadLetterPageSize = 100

// GetDeadLetterDeliveriesInput contains the pagination
// parameters supplied by the caller.
type GetDeadLetterDeliveriesInput struct {
	Limit  int
	Offset int
}

// GetDeadLetterDeliveriesOutput contains the DLQ records
// and pagination information.
type GetDeadLetterDeliveriesOutput struct {
	Deliveries []*domain.DeadLetterDelivery
	Limit      int
	Offset     int
	HasMore    bool
}

// GetDeadLetterDeliveriesUseCase retrieves failed deliveries
// that exhausted their retry attempts and entered the DLQ.
type GetDeadLetterDeliveriesUseCase struct {
	repository domain.NotificationRepository
}

// NewGetDeadLetterDeliveriesUseCase creates the use case.
func NewGetDeadLetterDeliveriesUseCase(
	repository domain.NotificationRepository,
) *GetDeadLetterDeliveriesUseCase {

	return &GetDeadLetterDeliveriesUseCase{
		repository: repository,
	}
}

// Execute retrieves one page of dead-lettered deliveries.
func (uc *GetDeadLetterDeliveriesUseCase) Execute(
	ctx context.Context,
	input GetDeadLetterDeliveriesInput,
) (*GetDeadLetterDeliveriesOutput, error) {

	// STEP 1: Validate repository dependency.
	if uc.repository == nil {
		return nil, fmt.Errorf(
			"notification repository is not configured",
		)
	}

	// STEP 2: Apply default page size.
	if input.Limit == 0 {
		input.Limit = defaultDeadLetterPageSize
	}

	// STEP 3: Validate pagination parameters.
	if input.Limit < 1 ||
		input.Limit > maxDeadLetterPageSize ||
		input.Offset < 0 {

		return nil, domain.ErrInvalidPagination
	}

	// STEP 4: Fetch one extra record.
	//
	// This helps us determine whether another page exists
	// without running a separate COUNT query.
	deliveries, err := uc.repository.GetDeadLetterDeliveries(
		ctx,
		input.Limit+1,
		input.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get dead-letter deliveries: %w",
			err,
		)
	}

	// STEP 5: Determine whether more records exist.
	hasMore := len(deliveries) > input.Limit

	// STEP 6: Remove the extra record.
	if hasMore {
		deliveries = deliveries[:input.Limit]
	}

	// STEP 7: Return the paginated result.
	return &GetDeadLetterDeliveriesOutput{
		Deliveries: deliveries,
		Limit:      input.Limit,
		Offset:     input.Offset,
		HasMore:    hasMore,
	}, nil
}
