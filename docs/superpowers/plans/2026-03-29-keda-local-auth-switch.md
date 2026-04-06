# KEDA Local Auth Switch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `dequeue` KEDA scaler use static AWS credentials in local kind development while preserving operator-based AWS identity for production.

**Architecture:** Parameterize `dequeue-app` chart authentication with two explicit modes, `secret` and `operator`. `values.yaml` keeps the production-oriented operator identity defaults, while `values/develop.yaml` switches the scaler to secret-backed credentials so `make install-dequeue` works against ElasticMQ on kind without changing the app container contract.

**Tech Stack:** Helm templates, Kubernetes, KEDA, kind, ElasticMQ

---

### Task 1: Add Values-Driven KEDA Auth Modes To `dequeue-app`

**Files:**
- Modify: `manifest/dequeue-app/values.yaml`
- Modify: `manifest/dequeue-app/values/develop.yaml`
- Modify: `manifest/dequeue-app/templates/scaledjob.yaml`
- Modify: `manifest/dequeue-app/templates/triggerauthentication.yaml`

- [ ] **Step 1: Write the failing checks**

Run:

```bash
env HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm template dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml
```

Expected: Template output still contains `identityOwner: operator` and `podIdentity:` even for develop, which proves the local auth mode is not switchable yet.

- [ ] **Step 2: Update values to declare auth mode**

Replace `manifest/dequeue-app/values.yaml` with:

```yaml
image:
  repository: dequeue
  tag: latest
  pullPolicy: IfNotPresent

queueName: sample-queue
queueURL: http://elasticmq.default.svc.cluster.local:9324/queue/sample-queue
queueLength: "1"
awsEndpoint: http://elasticmq.default.svc.cluster.local:9324
awsRegion: elasticmq
awsAccessKeyID: x
awsSecretAccessKey: x
dbConnectionString: postgres://app:app@postgresql.default.svc.cluster.local:5432/app?sslmode=disable
pollingInterval: 5
maxReplicaCount: 5
successfulJobsHistoryLimit: 2
failedJobsHistoryLimit: 2

kedaAuthentication:
  mode: operator
  podIdentityProvider: aws-eks
```

Replace `manifest/dequeue-app/values/develop.yaml` with:

```yaml
image:
  repository: local/dequeue
  tag: dev

kedaAuthentication:
  mode: secret
```

- [ ] **Step 3: Update templates to honor auth mode**

Replace `manifest/dequeue-app/templates/scaledjob.yaml` with:

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledJob
metadata:
  name: {{ include "dequeue.fullname" . }}
spec:
  pollingInterval: {{ .Values.pollingInterval }}
  maxReplicaCount: {{ .Values.maxReplicaCount }}
  successfulJobsHistoryLimit: {{ .Values.successfulJobsHistoryLimit }}
  failedJobsHistoryLimit: {{ .Values.failedJobsHistoryLimit }}
  jobTargetRef:
    template:
      spec:
        restartPolicy: Never
        containers:
          - name: dequeue
            image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
            imagePullPolicy: {{ .Values.image.pullPolicy }}
            env:
              - name: AWS_ENDPOINT
                value: {{ .Values.awsEndpoint | quote }}
              - name: AWS_REGION
                value: {{ .Values.awsRegion | quote }}
              - name: QUEUE_NAME
                value: {{ .Values.queueName | quote }}
              - name: QUEUE_URL
                value: {{ .Values.queueURL | quote }}
              - name: AWS_ACCESS_KEY_ID
                valueFrom:
                  secretKeyRef:
                    name: {{ include "dequeue.fullname" . }}-config
                    key: AWS_ACCESS_KEY_ID
              - name: AWS_SECRET_ACCESS_KEY
                valueFrom:
                  secretKeyRef:
                    name: {{ include "dequeue.fullname" . }}-config
                    key: AWS_SECRET_ACCESS_KEY
              - name: DB_CONNECTION_STRING
                valueFrom:
                  secretKeyRef:
                    name: {{ include "dequeue.fullname" . }}-config
                    key: DB_CONNECTION_STRING
  triggers:
    - type: aws-sqs-queue
      metadata:
        queueURLFromEnv: QUEUE_URL
        queueLength: {{ .Values.queueLength | quote }}
        awsRegion: {{ .Values.awsRegion | quote }}
        {{- if eq .Values.kedaAuthentication.mode "operator" }}
        identityOwner: operator
        {{- end }}
        awsEndpoint: {{ .Values.awsEndpoint | quote }}
      authenticationRef:
        name: {{ include "dequeue.fullname" . }}
```

Replace `manifest/dequeue-app/templates/triggerauthentication.yaml` with:

```yaml
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{ include "dequeue.fullname" . }}
spec:
  {{- if eq .Values.kedaAuthentication.mode "secret" }}
  secretTargetRef:
    - parameter: awsAccessKeyID
      name: {{ include "dequeue.fullname" . }}-config
      key: AWS_ACCESS_KEY_ID
    - parameter: awsSecretAccessKey
      name: {{ include "dequeue.fullname" . }}-config
      key: AWS_SECRET_ACCESS_KEY
  {{- else if eq .Values.kedaAuthentication.mode "operator" }}
  podIdentity:
    provider: {{ .Values.kedaAuthentication.podIdentityProvider | quote }}
  {{- else }}
  {{- fail (printf "unsupported kedaAuthentication.mode %q" .Values.kedaAuthentication.mode) }}
  {{- end }}
```

- [ ] **Step 4: Run template checks to verify both modes render**

Run:

```bash
env HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm template dequeue ./manifest/dequeue-app
env HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm template dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml
```

Expected:
- default render contains `identityOwner: operator` and `podIdentity`
- develop render contains `secretTargetRef` and does not contain `identityOwner: operator`

- [ ] **Step 5: Commit**

```bash
git add manifest/dequeue-app/values.yaml manifest/dequeue-app/values/develop.yaml manifest/dequeue-app/templates/scaledjob.yaml manifest/dequeue-app/templates/triggerauthentication.yaml
git commit -m "feat: support local keda auth for dequeue"
```

### Task 2: Document Local Versus Production KEDA Auth

**Files:**
- Modify: `README.md`
- Modify: `TODO.md`

- [ ] **Step 1: Write the failing documentation check**

Run:

```bash
rg -n "identityOwner: operator|aws-eks|static credential|secretTargetRef" README.md TODO.md
```

Expected: Documentation mentions production operator identity only and does not explain that develop uses static credential mode.

- [ ] **Step 2: Update README**

Update the KEDA auth section in `README.md` so it states:

```md
`make install-dequeue` は `manifest/dequeue-app/values/develop.yaml` を使い、KEDA scaler も `dequeue-config` Secret の static credential で ElasticMQ を参照します。kind / ElasticMQ 検証では追加の AWS identity 設定は不要です。

`manifest/keda-operator/values/production.yaml` は KEDA Operator 専用 ServiceAccount を作る前提です。production 向けの `dequeue` scaler は `identityOwner: operator` と `TriggerAuthentication.podIdentity.provider: aws-eks` を使うため、本番ではこの ServiceAccount に対して EKS Pod Identity Association を作成し、SQS 読み取り権限を付与します。
```

- [ ] **Step 3: Update TODO**

Update `TODO.md` so:

- the `make helm-deps` verification checks for `manifest/keda-operator/charts/keda-2.18.1.tgz`
- the KEDA verification notes make it explicit that develop uses static credential mode for the scaler
- the memo section states production alone uses operator identity

Use wording equivalent to:

```md
- [ ] `make helm-deps`
  - [ ] `test -f manifest/keda-operator/charts/keda-2.18.1.tgz`
```

and:

```md
- develop の `dequeue` chart は KEDA scaler も Secret の static credential で ElasticMQ を読む
- production の `dequeue` scaler は `identityOwner: operator` と `aws-eks` Pod Identity を使う
```

- [ ] **Step 4: Run the documentation check again**

Run:

```bash
rg -n "identityOwner: operator|aws-eks|static credential|secretTargetRef|keda-2.18.1" README.md TODO.md
```

Expected: Matches include both local static credential wording and production operator identity wording, plus the updated chart archive version.

- [ ] **Step 5: Commit**

```bash
git add README.md TODO.md
git commit -m "docs: describe local keda authentication"
```

### Task 3: Verify The Fix On kind

**Files:**
- Modify: none

- [ ] **Step 1: Reinstall the dequeue chart with develop values**

Run:

```bash
make install-dequeue
```

Expected: Helm upgrade succeeds for release `dequeue`.

- [ ] **Step 2: Verify the scaler is no longer failing on auth**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl describe scaledjob dequeue
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl logs -n keda deploy/keda-operator --tail=100
```

Expected:
- `kubectl describe scaledjob dequeue` no longer reports repeated `KEDAScalerFailed` due to IMDS credential lookup
- operator logs no longer show `no EC2 IMDS role found` for `dequeue`

- [ ] **Step 3: Verify KEDA creates jobs**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get jobs --watch
```

Expected: A `dequeue-*` Job appears after the scaler polls the queue. Stop the watch after confirming at least one job creation.

- [ ] **Step 4: Verify dequeue results reach PostgreSQL**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl port-forward svc/postgresql 5432:5432
psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
```

Expected: Query returns at least one row written by `dequeue`.

- [ ] **Step 5: Commit**

```bash
git commit --allow-empty -m "test: verify local keda dequeue scaling"
```
