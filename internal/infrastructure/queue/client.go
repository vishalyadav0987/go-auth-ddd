package queue

import (
	"github.com/hibiken/asynq"
)

func NewQueueClient(redisAddr string) *asynq.Client {
	return asynq.NewClient(
		asynq.RedisClientOpt{
			Addr: redisAddr,
		},
	)
}
