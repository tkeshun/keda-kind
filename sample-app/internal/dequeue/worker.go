package dequeue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"keda-kind/sample-app/internal/message"
	"keda-kind/sample-app/internal/timeutil"
)

type QueueMessage struct {
	Body          string
	ReceiptHandle string
}

type Queue interface {
	ReceiveOne(ctx context.Context, queueURL string, waitSeconds int32) (*QueueMessage, error)
	Delete(ctx context.Context, queueURL string, receiptHandle string) error
}

type StoredMessage struct {
	Code      string
	SentAt    time.Time
	StoredAt  time.Time
	RawBody   string
	QueueName string
}

type Store interface {
	Init(ctx context.Context) error
	Save(ctx context.Context, msg StoredMessage) error
}

type Worker struct {
	Queue       Queue
	Store       Store
	QueueURL    string
	QueueName   string
	WaitSeconds int32
	StoreDelay  time.Duration
	Now         func() time.Time
	Sleep       func(context.Context, time.Duration) error
}

func (w Worker) RunOnce(ctx context.Context) (bool, error) {
	if err := w.Store.Init(ctx); err != nil {
		return false, err
	}

	msg, err := w.Queue.ReceiveOne(ctx, w.QueueURL, w.WaitSeconds)
	if err != nil {
		return false, err
	}
	if msg == nil {
		return false, nil
	}

	var payload message.Payload
	if err := json.Unmarshal([]byte(msg.Body), &payload); err != nil {
		return false, err
	}

	now := time.Now()
	if w.Now != nil {
		now = w.Now()
	}
	now = timeutil.ToJST(now)

	sentAt := timeutil.ToJST(payload.SentAt)

	if w.StoreDelay > 0 {
		sleep := w.Sleep
		if sleep == nil {
			sleep = sleepContext
		}
		if err := sleep(ctx, w.StoreDelay); err != nil {
			return false, err
		}
	}

	if err := w.Store.Save(ctx, StoredMessage{
		Code:      payload.Code,
		SentAt:    sentAt,
		StoredAt:  now,
		RawBody:   msg.Body,
		QueueName: w.QueueName,
	}); err != nil {
		return false, err
	}

	if msg.ReceiptHandle == "" {
		return false, errors.New("receipt handle is required")
	}

	if err := w.Queue.Delete(ctx, w.QueueURL, msg.ReceiptHandle); err != nil {
		return false, err
	}

	return true, nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
