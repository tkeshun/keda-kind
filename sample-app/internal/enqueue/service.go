package enqueue

import (
	"context"
	"encoding/json"
	"time"

	"keda-kind/sample-app/internal/message"
)

type Queue interface {
	EnsureQueue(ctx context.Context, queueName string) (string, error)
	VisibleMessageCount(ctx context.Context, queueURL string) (int, error)
	SendMessage(ctx context.Context, queueURL string, body string) error
}

const maxVisibleMessages = 10

type TickResult struct {
	Sent    bool
	Skipped bool
}

type Service struct {
	Queue     Queue
	QueueName string
	Clock     func() time.Time
	RandIntn  func(int) int
}

func (s Service) Tick(ctx context.Context) (TickResult, error) {
	queueURL, err := s.Queue.EnsureQueue(ctx, s.QueueName)
	if err != nil {
		return TickResult{}, err
	}

	visibleCount, err := s.Queue.VisibleMessageCount(ctx, queueURL)
	if err != nil {
		return TickResult{}, err
	}
	if visibleCount >= maxVisibleMessages {
		return TickResult{Skipped: true}, nil
	}

	payload := message.Generate(s.Clock(), s.RandIntn)
	body, err := json.Marshal(payload)
	if err != nil {
		return TickResult{}, err
	}

	if err := s.Queue.SendMessage(ctx, queueURL, string(body)); err != nil {
		return TickResult{}, err
	}

	return TickResult{Sent: true}, nil
}
