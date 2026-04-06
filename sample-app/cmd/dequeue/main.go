package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"keda-kind/sample-app/internal/adapters/postgres"
	"keda-kind/sample-app/internal/adapters/sqs"
	"keda-kind/sample-app/internal/config"
	dequeueapp "keda-kind/sample-app/internal/dequeue"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoadDequeue()

	queueClient, err := sqs.New(ctx, cfg.AWS)
	if err != nil {
		log.Fatalf("create queue client: %v", err)
	}

	store, err := postgres.New(ctx, cfg.DBConnectionString)
	if err != nil {
		log.Fatalf("connect store: %v", err)
	}
	defer store.Close()

	worker := dequeueapp.Worker{
		Queue:       queueClient,
		Store:       store,
		QueueURL:    cfg.QueueURL,
		QueueName:   cfg.QueueName,
		WaitSeconds: cfg.WaitSeconds,
		StoreDelay:  cfg.StoreDelay,
		Now:         time.Now,
	}

	processed, err := worker.RunOnce(ctx)
	if err != nil {
		log.Fatalf("process queue item: %v", err)
	}
	if !processed {
		log.Print("no message available")
		return
	}

	log.Print("message stored successfully")
}
