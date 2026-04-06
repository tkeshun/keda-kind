# Enqueue Queue Capacity Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent the sample enqueue app from sending an 11th queue item by skipping sends when the queue already has 10 or more visible messages.

**Architecture:** Extend the enqueue queue abstraction with a queue-count lookup, make `enqueue.Service.Tick` return whether it sent or skipped, and update the SQS adapter plus CLI logging to reflect the new behavior. The behavior stays local to enqueue and is verified with enqueue-focused unit tests first.

**Tech Stack:** Go, AWS SDK v2 for SQS, Go testing package

---

### Task 1: Add Queue Capacity Guard To Enqueue

**Files:**
- Modify: `sample-app/internal/enqueue/service_test.go`
- Modify: `sample-app/internal/enqueue/service.go`
- Modify: `sample-app/internal/adapters/sqs/client.go`
- Modify: `sample-app/cmd/enqueue/main.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestServiceTickCreatesQueueAndPublishesPayloadWhenQueueHasCapacity(t *testing.T) {
	queue := &fakeQueue{messageCount: 9}
	now := time.Date(2026, 3, 29, 10, 10, 0, 0, time.UTC)

	service := Service{
		Queue:     queue,
		QueueName: "sample",
		Clock: func() time.Time { return now },
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
	if result.Skipped {
		t.Fatal("expected message to be sent")
	}
}

func TestServiceTickSkipsPublishWhenQueueAlreadyHasTenMessages(t *testing.T) {
	queue := &fakeQueue{messageCount: 10}

	service := Service{
		Queue:     queue,
		QueueName: "sample",
		Clock:     time.Now,
		RandIntn:  func(int) int { return 1 },
	}

	result, err := service.Tick(context.Background())
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	if !result.Skipped {
		t.Fatal("expected tick to skip enqueue")
	}
	if queue.body != "" {
		t.Fatalf("expected no message body, got %q", queue.body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./sample-app/internal/enqueue`
Expected: FAIL because `fakeQueue` and `Service.Tick` do not yet support queue counts or skip results

- [ ] **Step 3: Write minimal implementation**

```go
type Queue interface {
	EnsureQueue(ctx context.Context, queueName string) (string, error)
	MessageCount(ctx context.Context, queueURL string) (int, error)
	SendMessage(ctx context.Context, queueURL string, body string) error
}

type TickResult struct {
	Skipped bool
}

func (s Service) Tick(ctx context.Context) (TickResult, error) {
	queueURL, err := s.Queue.EnsureQueue(ctx, s.QueueName)
	if err != nil {
		return TickResult{}, err
	}

	count, err := s.Queue.MessageCount(ctx, queueURL)
	if err != nil {
		return TickResult{}, err
	}
	if count >= 10 {
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

	return TickResult{}, nil
}
```

```go
func (f *fakeQueue) MessageCount(_ context.Context, queueURL string) (int, error) {
	f.queueURL = queueURL
	return f.messageCount, nil
}
```

```go
func (c *Client) MessageCount(ctx context.Context, queueURL string) (int, error) {
	out, err := c.api.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &queueURL,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("get queue attributes: %w", err)
	}

	raw, ok := out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)]
	if !ok {
		return 0, fmt.Errorf("missing queue attribute: %s", types.QueueAttributeNameApproximateNumberOfMessages)
	}

	count, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse queue attribute %s: %w", types.QueueAttributeNameApproximateNumberOfMessages, err)
	}

	return count, nil
}
```

```go
	result, err := service.Tick(ctx)
	if err != nil {
		log.Fatalf("initial send failed: %v", err)
	}
	if result.Skipped {
		log.Printf("enqueue skipped for queue %q because it already has 10 or more messages", cfg.QueueName)
	} else {
		log.Printf("message sent to queue %q", cfg.QueueName)
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sample-app/internal/enqueue ./sample-app/internal/adapters/sqs ./sample-app/cmd/enqueue`
Expected: PASS for enqueue-related packages

- [ ] **Step 5: Run broader verification**

Run: `go test ./...`
Expected: PASS with no enqueue regressions

- [ ] **Step 6: Commit**

```bash
git add sample-app/internal/enqueue/service_test.go sample-app/internal/enqueue/service.go sample-app/internal/adapters/sqs/client.go sample-app/cmd/enqueue/main.go
git commit -m "feat: cap sample enqueue queue length"
```
