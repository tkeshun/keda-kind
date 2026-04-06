# Dequeue Store Delay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `dequeue-app` が DB 保存直前に環境変数で指定した秒数だけ待機できるようにする。

**Architecture:** `config.LoadDequeue` で `DEQUEUE_STORE_DELAY_SECONDS` を `time.Duration` に変換し、`cmd/dequeue` から `dequeue.Worker` に渡す。`Worker` は保存直前だけ待機処理を呼び、テストでは差し替え可能な sleep 関数で実時間待ちを避ける。Helm chart と compose には同じ環境変数を通す。

**Tech Stack:** Go, Go testing package, Helm template, Docker Compose

---

### Task 1: Add Configurable Store Delay To Dequeue

**Files:**
- Modify: `sample-app/internal/config/config_test.go`
- Modify: `sample-app/internal/dequeue/worker_test.go`
- Modify: `sample-app/internal/config/config.go`
- Modify: `sample-app/internal/dequeue/worker.go`
- Modify: `sample-app/cmd/dequeue/main.go`
- Modify: `manifest/dequeue-app/values.yaml`
- Modify: `manifest/dequeue-app/templates/scaledjob.yaml`
- Modify: `compose.yaml`

- [ ] **Step 1: Write the failing tests**

```go
func TestLoadDequeueReadsStoreDelaySeconds(t *testing.T) {
	cfg, err := LoadDequeue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "DB_CONNECTION_STRING":
			return "postgres://example"
		case "DEQUEUE_STORE_DELAY_SECONDS":
			return "5"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.StoreDelay != 5*time.Second {
		t.Fatalf("unexpected store delay: %s", cfg.StoreDelay)
	}
}

func TestLoadDequeueRejectsInvalidStoreDelaySeconds(t *testing.T) {
	_, err := LoadDequeue(func(key string) string {
		switch key {
		case "AWS_ENDPOINT":
			return "http://elasticmq:9324"
		case "DB_CONNECTION_STRING":
			return "postgres://example"
		case "DEQUEUE_STORE_DELAY_SECONDS":
			return "bad"
		default:
			return ""
		}
	})
	if err == nil {
		t.Fatal("expected an error when DEQUEUE_STORE_DELAY_SECONDS is invalid")
	}
}

func TestWorkerRunOnceSleepsBeforeSavingMessage(t *testing.T) {
	queue := &fakeQueueClient{
		message: &QueueMessage{
			Body:          `{"code":"54321","sent_at":"2026-03-29T10:15:00+09:00"}`,
			ReceiptHandle: "receipt-1",
		},
	}
	store := &fakeStore{}
	var slept time.Duration

	worker := Worker{
		Queue:       queue,
		Store:       store,
		QueueURL:    "http://elasticmq:9324/queue/sample",
		QueueName:   "sample",
		WaitSeconds: 1,
		StoreDelay:  5 * time.Second,
		Sleep: func(_ context.Context, d time.Duration) error {
			slept = d
			return nil
		},
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once failed: %v", err)
	}
	if !processed {
		t.Fatal("expected one message to be processed")
	}
	if slept != 5*time.Second {
		t.Fatalf("unexpected sleep duration: %s", slept)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./sample-app/internal/config ./sample-app/internal/dequeue`
Expected: FAIL because `config.Dequeue` and `dequeue.Worker` do not yet support store delay

- [ ] **Step 3: Write the minimal implementation**

```go
type Dequeue struct {
	AWS                AWS
	QueueName          string
	QueueURL           string
	DBConnectionString string
	WaitSeconds        int32
	StoreDelay         time.Duration
}
```

```go
	storeDelay := time.Duration(0)
	if raw := getenv("DEQUEUE_STORE_DELAY_SECONDS"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return Dequeue{}, fmt.Errorf("parse DEQUEUE_STORE_DELAY_SECONDS: %w", err)
		}
		if parsed < 0 {
			return Dequeue{}, errors.New("DEQUEUE_STORE_DELAY_SECONDS must be >= 0")
		}
		storeDelay = time.Duration(parsed) * time.Second
	}
```

```go
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
```

```go
	if w.StoreDelay > 0 {
		sleep := w.Sleep
		if sleep == nil {
			sleep = sleepContext
		}
		if err := sleep(ctx, w.StoreDelay); err != nil {
			return false, err
		}
	}
```

```go
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
```

```go
	worker := dequeueapp.Worker{
		Queue:       queueClient,
		Store:       store,
		QueueURL:    cfg.QueueURL,
		QueueName:   cfg.QueueName,
		WaitSeconds: cfg.WaitSeconds,
		StoreDelay:  cfg.StoreDelay,
		Now:         time.Now,
	}
```

```yaml
storeDelaySeconds: "0"
```

```yaml
              - name: DEQUEUE_STORE_DELAY_SECONDS
                value: {{ .Values.storeDelaySeconds | quote }}
```

```yaml
      DEQUEUE_STORE_DELAY_SECONDS: "0"
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./sample-app/internal/config ./sample-app/internal/dequeue`
Expected: PASS

- [ ] **Step 5: Run broader verification**

Run: `go test ./...`
Expected: PASS with no dequeue regressions

- [ ] **Step 6: Commit**

```bash
git add sample-app/internal/config/config_test.go sample-app/internal/dequeue/worker_test.go sample-app/internal/config/config.go sample-app/internal/dequeue/worker.go sample-app/cmd/dequeue/main.go manifest/dequeue-app/values.yaml manifest/dequeue-app/templates/scaledjob.yaml compose.yaml docs/superpowers/specs/2026-04-03-dequeue-store-delay-design.md docs/superpowers/plans/2026-04-03-dequeue-store-delay.md
git commit -m "feat: add configurable dequeue store delay"
```
