package notification

import (
	"math/rand"
	"time"

	domain "github.com/vishalyadav0987/authentication/internal/domain/notification"
)

const (
	baseRetryDelay = 10 * time.Second
	maxRetryDelay  = 15 * time.Minute
)

func calculateRetryDelay(attemptNumber int) time.Duration {
	if attemptNumber < 1 {
		attemptNumber = 1
	}

	delay := baseRetryDelay

	for i := 1; i < attemptNumber; i++ {
		delay *= 2

		if delay >= maxRetryDelay {
			delay = maxRetryDelay
			break
		}
	}

	// Full jitter: choose a random delay between zero and the
	// calculated exponential backoff duration.
	if delay <= 0 {
		return baseRetryDelay
	}

	return time.Duration(rand.Int63n(int64(delay) + 1))
}

// This produces a randomized delay between zero and the exponential backoff limit.

// For example:
// Attempt	Maximum delay	Actual delay
// 1	10 sec	Random
// 2	20 sec	Random
// 3	40 sec	Random
// 4	80 sec	Random
// 5	160 sec	Random

// The maximum remains capped at 15 minutes.

func calculateProviderRetryDelay(
	attemptNumber int,
	err error,
) time.Duration {
	delay := calculateRetryDelay(attemptNumber)

	providerErr, ok := domain.IsProviderError(err)
	if !ok {
		return delay
	}

	if providerErr.RetryAfter > delay {
		return providerErr.RetryAfter
	}

	return delay
}
