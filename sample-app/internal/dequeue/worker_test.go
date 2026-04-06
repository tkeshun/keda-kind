package dequeue

import (
	"context"
	"errors"
	"testing"
	"time"

	"keda-kind/sample-app/internal/timeutil"
)

var (
	errUnexpectedQueueURL    = errors.New("unexpected queue URL")
	errUnexpectedWaitSeconds = errors.New("unexpected wait seconds")
)

type fakeQueueClient struct {
	message       *QueueMessage
	deletedHandle string
}

func (f *fakeQueueClient) ReceiveOne(_ context.Context, queueURL string, waitSeconds int32) (*QueueMessage, error) {
	if queueURL != "http://elasticmq:9324/queue/sample" {
		return nil, errUnexpectedQueueURL
	}
	if waitSeconds != 1 {
		return nil, errUnexpectedWaitSeconds
	}
	return f.message, nil
}

func (f *fakeQueueClient) Delete(_ context.Context, queueURL string, receiptHandle string) error {
	if queueURL != "http://elasticmq:9324/queue/sample" {
		return errUnexpectedQueueURL
	}
	f.deletedHandle = receiptHandle
	return nil
}

type fakeStore struct {
	initCalled bool
	saved      StoredMessage
	events     *[]string
}

func (f *fakeStore) Init(_ context.Context) error {
	f.initCalled = true
	return nil
}

func (f *fakeStore) Save(_ context.Context, msg StoredMessage) error {
	if f.events != nil {
		*f.events = append(*f.events, "save")
	}
	f.saved = msg
	return nil
}

func TestWorkerRunOnceStoresAndDeletesOneMessage(t *testing.T) {
	queue := &fakeQueueClient{
		message: &QueueMessage{
			Body:          `{"code":"54321","sent_at":"2026-03-29T10:15:00+09:00"}`,
			ReceiptHandle: "receipt-1",
		},
	}
	store := &fakeStore{}
	worker := Worker{
		Queue:       queue,
		Store:       store,
		QueueURL:    "http://elasticmq:9324/queue/sample",
		WaitSeconds: 1,
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC)
		},
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once failed: %v", err)
	}
	if !processed {
		t.Fatal("expected one message to be processed")
	}
	if !store.initCalled {
		t.Fatal("expected store init to be called")
	}
	if store.saved.Code != "54321" {
		t.Fatalf("unexpected saved code: %s", store.saved.Code)
	}
	expectedTime := time.Date(2026, 3, 29, 10, 15, 0, 0, timeutil.JSTLocation())
	if !store.saved.SentAt.Equal(expectedTime) {
		t.Fatalf("unexpected saved time: %s", store.saved.SentAt)
	}
	expectedStoredAt := timeutil.ToJST(time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC))
	if !store.saved.StoredAt.Equal(expectedStoredAt) {
		t.Fatalf("unexpected stored time: %s", store.saved.StoredAt)
	}
	if queue.deletedHandle != "receipt-1" {
		t.Fatalf("unexpected deleted handle: %s", queue.deletedHandle)
	}
}

func TestWorkerRunOnceReturnsFalseWhenQueueIsEmpty(t *testing.T) {
	worker := Worker{
		Queue:       &fakeQueueClient{},
		Store:       &fakeStore{},
		QueueURL:    "http://elasticmq:9324/queue/sample",
		WaitSeconds: 1,
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once failed: %v", err)
	}
	if processed {
		t.Fatal("expected queue to be empty")
	}
}

func TestWorkerRunOnceSleepsBeforeSavingMessage(t *testing.T) {
	queue := &fakeQueueClient{
		message: &QueueMessage{
			Body:          `{"code":"54321","sent_at":"2026-03-29T10:15:00+09:00"}`,
			ReceiptHandle: "receipt-1",
		},
	}
	events := []string{}
	store := &fakeStore{events: &events}
	worker := Worker{
		Queue:       queue,
		Store:       store,
		QueueURL:    "http://elasticmq:9324/queue/sample",
		QueueName:   "sample",
		WaitSeconds: 1,
		StoreDelay:  5 * time.Second,
		Sleep: func(_ context.Context, d time.Duration) error {
			events = append(events, "sleep:"+d.String())
			return nil
		},
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC)
		},
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once failed: %v", err)
	}
	if !processed {
		t.Fatal("expected one message to be processed")
	}
	if len(events) != 2 {
		t.Fatalf("unexpected events: %#v", events)
	}
	if events[0] != "sleep:5s" {
		t.Fatalf("expected sleep before save, got %#v", events)
	}
	if events[1] != "save" {
		t.Fatalf("expected save after sleep, got %#v", events)
	}
}

func TestWorkerRunOnceDoesNotSleepWhenStoreDelayIsZero(t *testing.T) {
	queue := &fakeQueueClient{
		message: &QueueMessage{
			Body:          `{"code":"54321","sent_at":"2026-03-29T10:15:00+09:00"}`,
			ReceiptHandle: "receipt-1",
		},
	}
	store := &fakeStore{}
	slept := false
	worker := Worker{
		Queue:       queue,
		Store:       store,
		QueueURL:    "http://elasticmq:9324/queue/sample",
		QueueName:   "sample",
		WaitSeconds: 1,
		StoreDelay:  0,
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = true
			return nil
		},
		Now: func() time.Time {
			return time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC)
		},
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once failed: %v", err)
	}
	if !processed {
		t.Fatal("expected one message to be processed")
	}
	if slept {
		t.Fatal("expected no sleep when store delay is zero")
	}
}
