package enqueue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"keda-kind/sample-app/internal/message"
)

type fakeQueue struct {
	ensuredName  string
	queueURL     string
	visibleCount int
	body         string
}

func (f *fakeQueue) EnsureQueue(_ context.Context, queueName string) (string, error) {
	f.ensuredName = queueName
	return "http://elasticmq:9324/queue/sample", nil
}

func (f *fakeQueue) VisibleMessageCount(_ context.Context, queueURL string) (int, error) {
	f.queueURL = queueURL
	return f.visibleCount, nil
}

func (f *fakeQueue) SendMessage(_ context.Context, queueURL string, body string) error {
	f.queueURL = queueURL
	f.body = body
	return nil
}

func TestServiceTickSendsWhenQueueHasNineVisibleMessages(t *testing.T) {
	queue := &fakeQueue{}
	queue.visibleCount = 9
	now := time.Date(2026, 3, 29, 10, 10, 0, 0, time.UTC)

	service := Service{
		Queue:     queue,
		QueueName: "sample",
		Clock: func() time.Time {
			return now
		},
		RandIntn: func(max int) int {
			if max != 90000 {
				t.Fatalf("unexpected max: %d", max)
			}
			return 7
		},
	}

	result, err := service.Tick(context.Background())
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	if !result.Sent {
		t.Fatalf("expected tick to send a message")
	}
	if result.Skipped {
		t.Fatalf("expected tick not to skip a message")
	}

	if queue.ensuredName != "sample" {
		t.Fatalf("unexpected queue name: %s", queue.ensuredName)
	}
	if queue.queueURL != "http://elasticmq:9324/queue/sample" {
		t.Fatalf("unexpected queue URL: %s", queue.queueURL)
	}
	if queue.body == "" {
		t.Fatalf("expected message body to be sent")
	}

	var payload message.Payload
	if err := json.Unmarshal([]byte(queue.body), &payload); err != nil {
		t.Fatalf("message body is not JSON: %v", err)
	}
	if payload.Code != "10007" {
		t.Fatalf("unexpected payload code: %s", payload.Code)
	}
	if !payload.SentAt.Equal(now) {
		t.Fatalf("unexpected sentAt: %s", payload.SentAt)
	}
}

func TestServiceTickSkipsWhenQueueHasTenVisibleMessages(t *testing.T) {
	queue := &fakeQueue{}
	queue.visibleCount = 10

	service := Service{
		Queue:     queue,
		QueueName: "sample",
		Clock: func() time.Time {
			return time.Unix(0, 0).UTC()
		},
		RandIntn: func(max int) int {
			t.Fatalf("RandIntn should not be called when enqueue is skipped")
			return 0
		},
	}

	result, err := service.Tick(context.Background())
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected tick to skip a message")
	}
	if result.Sent {
		t.Fatalf("expected tick not to send a message")
	}
	if queue.body != "" {
		t.Fatalf("expected no message body to be sent")
	}
}
