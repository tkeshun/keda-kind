package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"keda-kind/sample-app/internal/adapters/sqs"
	"keda-kind/sample-app/internal/config"
	enqueueapp "keda-kind/sample-app/internal/enqueue"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoadEnqueue()

	queueClient, err := sqs.New(ctx, cfg.AWS)
	if err != nil {
		log.Fatalf("create queue client: %v", err)
	}

	service := enqueueapp.Service{
		Queue:     queueClient,
		QueueName: cfg.QueueName,
		Clock:     time.Now,
		RandIntn:  rand.Intn,
	}

	switch cfg.Mode {
	case "http":
		if err := runHTTPServer(ctx, cfg, service); err != nil {
			log.Fatalf("http server failed: %v", err)
		}
	case "scheduled":
		runScheduled(ctx, cfg, service)
	default:
		log.Fatalf("unsupported enqueue mode: %s", cfg.Mode)
	}
}

func runScheduled(ctx context.Context, cfg config.Enqueue, service enqueueapp.Service) {
	if result, err := service.Tick(ctx); err != nil {
		log.Fatalf("initial send failed: %v", err)
	} else if result.Skipped {
		log.Printf("initial enqueue skipped for queue %q because it already has 10 or more messages", cfg.QueueName)
	} else {
		log.Printf("message sent to queue %q", cfg.QueueName)
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Print("shutdown requested")
			return
		case <-ticker.C:
			result, err := service.Tick(ctx)
			if err != nil {
				log.Printf("send failed: %v", err)
				continue
			}
			if result.Skipped {
				log.Printf("enqueue skipped for queue %q because it already has 10 or more messages", cfg.QueueName)
				continue
			}
			log.Printf("message sent to queue %q", cfg.QueueName)
		}
	}
}

type tickService interface {
	Tick(context.Context) (enqueueapp.TickResult, error)
}

func runHTTPServer(ctx context.Context, cfg config.Enqueue, service tickService) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: newHTTPHandler(service),
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("starting enqueue http server on :%d", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}

func newHTTPHandler(service tickService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /enqueue", func(w http.ResponseWriter, r *http.Request) {
		result, err := service.Tick(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
			return
		}
		if result.Skipped {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "skipped"})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}
