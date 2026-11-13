package queue

import (
	"log"

	"github.com/hibiken/asynq"
)

func StartWorker(redisAddr string) error {
	server := NewWorkerServer(redisAddr)

	mux := asynq.NewServeMux()

	orderEventHandler := NewOrderEventHandler()

	mux.HandleFunc(
		TaskTypeOrderEvent,
		orderEventHandler.HandleOrderEvent,
	)

	log.Println("Order event worker started...")

	return server.Run(mux)
}
