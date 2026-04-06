# Default AWS Credential Chain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the sample enqueue/dequeue apps rely on the AWS SDK v2 default credential chain instead of hard-coded static credentials.

**Architecture:** Remove access-key fields from `config.AWS`, keep endpoint and region as explicit config, and let the SQS adapter call `awsconfig.LoadDefaultConfig` without a custom credentials provider. Local chart Secrets remain as the source of env vars for ElasticMQ, while production can use Pod Identity through the same application code path.

**Tech Stack:** Go, AWS SDK v2, Helm, Kubernetes, KEDA, kind

---

### Task 1: Remove Static Credential Fields From App Config

**Files:**
- Modify: `sample-app/internal/config/config.go`
- Modify: `sample-app/internal/config/config_test.go`

- [ ] **Step 1: Write the failing test check**

Run:

```bash
env GOPATH=$(pwd)/.cache/go GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./sample-app/internal/config
```

Expected: Existing tests still encode `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` assumptions, so they must be updated together with the config change.

- [ ] **Step 2: Update config types and loaders**

Change `sample-app/internal/config/config.go` so `AWS` becomes:

```go
type AWS struct {
	Endpoint string
	Region   string
}
```

Update `loadAWS` to:

```go
func loadAWS(getenv func(string) string) (AWS, error) {
	endpoint := getenv("AWS_ENDPOINT")
	if endpoint == "" {
		return AWS{}, errors.New("AWS_ENDPOINT is required")
	}

	return AWS{
		Endpoint: endpoint,
		Region:   firstNonEmpty(getenv("AWS_REGION"), "elasticmq"),
	}, nil
}
```

Do not read `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY` in config anymore.

- [ ] **Step 3: Update config tests**

Adjust `sample-app/internal/config/config_test.go` so the env fixtures no longer need access key fields. The tests should still verify:

- `AWS_ENDPOINT` is required
- `AWS_REGION` defaults to `elasticmq`
- `QUEUE_NAME` and interval defaults still behave as before
- `DB_CONNECTION_STRING` remains required for dequeue

Keep the test file focused on config behavior only.

- [ ] **Step 4: Re-run config tests**

Run:

```bash
env GOPATH=$(pwd)/.cache/go GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./sample-app/internal/config
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add sample-app/internal/config/config.go sample-app/internal/config/config_test.go
git commit -m "refactor: remove static aws credentials from config"
```

### Task 2: Switch The SQS Adapter To The Default Credential Chain

**Files:**
- Modify: `sample-app/internal/adapters/sqs/client.go`

- [ ] **Step 1: Write the failing code check**

Run:

```bash
rg -n "StaticCredentialsProvider|WithCredentialsProvider|AccessKeyID|SecretAccessKey" sample-app/internal/adapters/sqs/client.go
```

Expected: The adapter still explicitly configures a static credentials provider.

- [ ] **Step 2: Update the SQS client**

Modify `sample-app/internal/adapters/sqs/client.go` so `New` becomes:

```go
func New(ctx context.Context, cfg config.AWS) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	api := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
		o.BaseEndpoint = &cfg.Endpoint
	})

	return &Client{api: api}, nil
}
```

Also remove the now-unused `credentials` import.

- [ ] **Step 3: Verify the adapter no longer pins credentials**

Run:

```bash
rg -n "StaticCredentialsProvider|WithCredentialsProvider|AccessKeyID|SecretAccessKey" sample-app/internal/adapters/sqs/client.go sample-app/internal/config/config.go
```

Expected: No matches in those two files.

- [ ] **Step 4: Re-run the full Go test suite**

Run:

```bash
env GOPATH=$(pwd)/.cache/go GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add sample-app/internal/adapters/sqs/client.go
git commit -m "refactor: use default aws credential chain"
```

### Task 3: Update Docs And Verify End-To-End On kind

**Files:**
- Modify: `README.md`
- Modify: `TODO.md`

- [ ] **Step 1: Update local-vs-production auth wording**

Adjust `README.md` and `TODO.md` so they state:

- the app code uses the AWS SDK default credential chain
- local ElasticMQ works because chart Secrets populate `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`
- production can rely on Pod Identity for the app and scaler without code branches

Keep the wording concise and aligned with the current chart layout.

- [ ] **Step 2: Verify chart renders still work**

Run:

```bash
env HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm template enqueue ./manifest/enqueue-app
env HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm template dequeue ./manifest/dequeue-app
env HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm template dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml
```

Expected: All renders succeed.

- [ ] **Step 3: Verify kind deployment still works**

Run:

```bash
make install-enqueue
make install-dequeue
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get jobs --watch
```

Expected: `dequeue-*` Jobs appear. Stop the watch after confirming at least one job.

- [ ] **Step 4: Verify PostgreSQL persistence**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl port-forward svc/postgresql 5432:5432
psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
```

Expected: Query returns rows written by `dequeue`.

- [ ] **Step 5: Commit**

```bash
git add README.md TODO.md
git commit -m "docs: describe aws credential chain behavior"
git commit --allow-empty -m "test: verify credential chain flow on kind"
```
