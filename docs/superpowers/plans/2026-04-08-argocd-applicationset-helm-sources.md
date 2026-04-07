# ArgoCD ApplicationSet Helm Source 移行実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 既存の静的 `Application` 3件を、`ApplicationSet` が Helm chart 参照の `Application` として生成する構成へ移行する。

**Architecture:** `argocd/applicationsets/env-bundle.yaml` を追加し、list generator で `develop` の in-cluster 向けに `keda-operator`、`infra-core`、`sample-app` の `Application` を生成する。生成された各 `Application` は既存どおり `manifest/keda-operator`、`manifest/infra-bundle`、`manifest/app-bundle` の Helm chart を `source.path` で参照する。

**Tech Stack:** Argo CD ApplicationSet、Argo CD Application、Helm chart、Kubernetes manifest、Makefile/kubectl 検証。

---

### Task 1: ApplicationSet 定義を追加する

**Files:**
- Create: `argocd/applicationsets/env-bundle.yaml`

- [x] **Step 1: `ApplicationSet` YAMLを作成する**

`keda-operator`、`infra-core`、`sample-app` の3要素をlist generatorに定義し、templateで `Application` を生成する。

- [x] **Step 2: YAML構文を確認する**

Run:

```bash
python3 -c 'import yaml; yaml.safe_load(open("argocd/applicationsets/env-bundle.yaml")); print("yaml ok")'
```

Expected: `yaml ok`

- [ ] **Step 3: CRD導入後にserver dry-runを確認する**

Run:

```bash
kubectl apply --dry-run=server -f argocd/applicationsets/env-bundle.yaml
```

Expected: `applicationset.argoproj.io/env-bundle created (server dry run)`

### Task 2: 旧静的Application YAMLと参照箇所を整理する

**Files:**
- Delete: `argocd/applications/keda-operator.yaml`
- Delete: `argocd/applications/infra-core.yaml`
- Delete: `argocd/applications/sample-app.yaml`
- Modify: `Makefile`
- Modify: `README.md`
- Modify: `TODO.md`
- Modify: `sample-app/layout/layout_test.go`

- [x] **Step 1: 旧静的Application YAMLを削除する**

`argocd/applications/*.yaml` を削除し、同名Applicationの二重管理を避ける。

- [x] **Step 2: `install-argocd-apps`をApplicationSet applyへ変更する**

Run target behavior:

```bash
env $(KUBE_ENV) kubectl apply -f argocd/applicationsets/env-bundle.yaml
```

- [x] **Step 3: README/TODO/layout testをApplicationSet前提へ更新する**

`argocd/applicationsets/env-bundle.yaml` を正のArgoCD制御定義として参照する。

### Task 3: 最終検証を行う

**Files:**
- Test: `argocd/applicationsets/env-bundle.yaml`
- Test: `manifest/keda-operator/Chart.yaml`
- Test: `manifest/infra-bundle/Chart.yaml`
- Test: `manifest/app-bundle/Chart.yaml`

- [x] **Step 1: Helm chart単体のrenderを確認する**

Run:

```bash
helm template keda-operator manifest/keda-operator -f manifest/keda-operator/values/develop.yaml
helm template infra-core manifest/infra-bundle -f manifest/infra-bundle/values/develop.yaml
helm template sample-app manifest/app-bundle -f manifest/app-bundle/values/develop.yaml
```

Expected: 3コマンドすべてがexit code 0で完了する。

- [x] **Step 2: レイアウト保護テストを実行する**

Run:

```bash
env GOPATH=$(pwd)/.cache/go GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./sample-app/layout
```

Expected: `ok`

- [x] **Step 3: 全体テストを実行する**

Run:

```bash
make test
```

Expected: `go test ./...` がexit code 0で完了する。

---

## Self-Review

- 既存の3つの `Application` の `source.path`、`values/develop.yaml`、destination namespaceをApplicationSetへ反映した。
- repo上の正をApplicationSetへ移すため、旧静的Application YAMLを削除した。
- README、TODO、layout test、Makefileの参照をApplicationSetへ更新した。
- CRD未導入環境ではkubectl dry-runが失敗するため、YAML parserで構文確認し、server dry-runはArgoCD導入後の残タスクとして明記した。
