# ArgoCD ApplicationSet Any Namespace 化実装計画

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `env-bundle` ApplicationSet と生成される Application を `sample-applicationset` namespace に移し、Argo CD から安全に reconcile できる構成にする。

**Architecture:** Argo CD 本体は引き続き `argocd` namespace に置き、ApplicationSet / Application の置き場所だけを `sample-applicationset` に分離する。Argo CD の `application.namespaces` と `applicationsetcontroller.namespaces` を有効化し、専用 `AppProject` で `sample-applicationset` からの Application 利用を許可する。

**Tech Stack:** Argo CD ApplicationSet in any namespace、Argo CD Application in any namespace、AppProject、Helm chart values、Kubernetes manifest、Makefile/kubectl/go test 検証。

---

## File Structure

- Modify: `manifest/argocd/values.yaml`
  - Argo CD cmd params に `application.namespaces` と `applicationsetcontroller.namespaces` を追加する。
- Create: `argocd/namespaces/sample-applicationset.yaml`
  - `sample-applicationset` namespace を宣言的に作る。
- Create: `argocd/projects/sample-app.yaml`
  - `sample-applicationset` namespace の Application が参照する専用 AppProject を作る。
- Modify: `argocd/applicationsets/env-bundle.yaml`
  - ApplicationSet と生成 Application の namespace を `sample-applicationset` に変更し、`project` を `sample-app` に変更する。
- Modify: `Makefile`
  - `install-argocd-apps` で namespace、AppProject、ApplicationSet の順に apply する。
- Modify: `README.md`
  - Any namespace 構成、実行 context、確認コマンドを説明する。
- Modify: `TODO.md`
  - `kubectl get` / `describe` の namespace を `sample-applicationset` に更新する。
- Modify: `sample-app/layout/layout_test.go`
  - 新規 namespace / AppProject manifest と README/Makefile の参照を保護する。

---

### Task 1: レイアウト保護テストを先に更新する

**Files:**
- Modify: `sample-app/layout/layout_test.go`

- [ ] **Step 1: 失敗するテストを書く**

`TestSampleAppLayoutUsesSampleAppPaths` の `requiredFiles` に以下を追加する。

```go
"argocd/namespaces/sample-applicationset.yaml",
"argocd/projects/sample-app.yaml",
```

`TestMakefileReferencesSampleAppAssets` の `requiredSnippets` に以下を追加する。

```go
"argocd/namespaces/sample-applicationset.yaml",
"argocd/projects/sample-app.yaml",
```

`TestReadmeMentionsArgoCDFlow` の `requiredSnippets` に以下を追加する。

```go
"sample-applicationset",
"argocd/projects/sample-app.yaml",
```

- [ ] **Step 2: テストが失敗することを確認する**

Run:

```bash
env GOPATH=$(pwd)/.cache/go GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./sample-app/layout
```

Expected: `argocd/namespaces/sample-applicationset.yaml` または `argocd/projects/sample-app.yaml` が存在しないため FAIL。

- [ ] **Step 3: コミットする**

```bash
git add sample-app/layout/layout_test.go
git commit -m "test: cover argocd applicationset namespace manifests"
```

---

### Task 2: Argo CD の any namespace 設定を追加する

**Files:**
- Modify: `manifest/argocd/values.yaml`

- [ ] **Step 1: `manifest/argocd/values.yaml` を更新する**

`configs.params` を以下の内容にする。

```yaml
argo-cd:
  fullnameOverride: argocd
  dex:
    enabled: false
  configs:
    params:
      server.insecure: true
      application.namespaces: sample-applicationset
      applicationsetcontroller.namespaces: sample-applicationset
  applicationSet:
    allowAnyNamespace: true
  server:
    service:
      type: ClusterIP
```

- [ ] **Step 2: Helm template に設定が出ることを確認する**

Run:

```bash
helm template argocd ./manifest/argocd -f manifest/argocd/values/develop.yaml -n argocd | rg -n "application.namespaces|applicationsetcontroller.namespaces|kind: ClusterRole|name: argocd-applicationset-controller"
```

Expected: `application.namespaces: sample-applicationset`、`applicationsetcontroller.namespaces: sample-applicationset`、ApplicationSet controller 用 `ClusterRole` が表示される。

- [ ] **Step 3: コミットする**

```bash
git add manifest/argocd/values.yaml
git commit -m "feat: enable argocd appset namespace"
```

---

### Task 3: namespace と専用 AppProject を追加する

**Files:**
- Create: `argocd/namespaces/sample-applicationset.yaml`
- Create: `argocd/projects/sample-app.yaml`

- [ ] **Step 1: `sample-applicationset` namespace manifest を作成する**

`argocd/namespaces/sample-applicationset.yaml` を作成する。

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: sample-applicationset
  labels:
    app.kubernetes.io/part-of: keda-kind
    argocd.argoproj.io/managed-by: argocd
```

- [ ] **Step 2: 専用 AppProject manifest を作成する**

`argocd/projects/sample-app.yaml` を作成する。

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: sample-app
  namespace: argocd
spec:
  description: Project for keda-kind sample applications managed from sample-applicationset.
  sourceRepos:
    - https://example.invalid/replace-with-your-git-remote.git
    - https://github.com/tkeshun/keda-kind.git
  sourceNamespaces:
    - sample-applicationset
  destinations:
    - server: https://kubernetes.default.svc
      namespace: default
    - server: https://kubernetes.default.svc
      namespace: keda
    - server: https://kubernetes.default.svc
      namespace: kube-system
  clusterResourceWhitelist:
    - group: ""
      kind: Namespace
    - group: admissionregistration.k8s.io
      kind: ValidatingWebhookConfiguration
    - group: apiextensions.k8s.io
      kind: CustomResourceDefinition
    - group: apiregistration.k8s.io
      kind: APIService
    - group: rbac.authorization.k8s.io
      kind: ClusterRole
    - group: rbac.authorization.k8s.io
      kind: ClusterRoleBinding
  namespaceResourceWhitelist:
    - group: ""
      kind: ConfigMap
    - group: ""
      kind: Secret
    - group: ""
      kind: Service
    - group: ""
      kind: ServiceAccount
    - group: apps
      kind: Deployment
    - group: keda.sh
      kind: ScaledJob
    - group: keda.sh
      kind: TriggerAuthentication
    - group: rbac.authorization.k8s.io
      kind: Role
    - group: rbac.authorization.k8s.io
      kind: RoleBinding
```

- [ ] **Step 3: YAML構文を確認する**

Run:

```bash
python3 -c 'import yaml; [yaml.safe_load(open(path)) for path in ["argocd/namespaces/sample-applicationset.yaml", "argocd/projects/sample-app.yaml"]]; print("yaml ok")'
```

Expected: `yaml ok`

- [ ] **Step 4: コミットする**

```bash
git add argocd/namespaces/sample-applicationset.yaml argocd/projects/sample-app.yaml
git commit -m "feat: add sample app argocd project"
```

---

### Task 4: ApplicationSet を `sample-applicationset` に移す

**Files:**
- Modify: `argocd/applicationsets/env-bundle.yaml`

- [ ] **Step 1: ApplicationSet と生成 Application の namespace を変更する**

`argocd/applicationsets/env-bundle.yaml` の該当箇所を以下にする。

```yaml
metadata:
  name: env-bundle
  namespace: sample-applicationset
```

```yaml
  template:
    metadata:
      name: "{{ .name }}"
      namespace: sample-applicationset
```

- [ ] **Step 2: 生成 Application の project を変更する**

`spec.template.spec.project` を以下にする。

```yaml
    spec:
      project: sample-app
```

- [ ] **Step 3: YAML構文を確認する**

Run:

```bash
python3 -c 'import yaml; yaml.safe_load(open("argocd/applicationsets/env-bundle.yaml")); print("yaml ok")'
```

Expected: `yaml ok`

- [ ] **Step 4: コミットする**

```bash
git add argocd/applicationsets/env-bundle.yaml
git commit -m "feat: move applicationset to sample namespace"
```

---

### Task 5: apply 導線とドキュメントを更新する

**Files:**
- Modify: `Makefile`
- Modify: `README.md`
- Modify: `TODO.md`

- [ ] **Step 1: `install-argocd-apps` を順序付き apply に変更する**

`Makefile` の `install-argocd-apps` を以下にする。

```make
install-argocd-apps:
	env $(KUBE_ENV) kubectl apply -f argocd/namespaces/sample-applicationset.yaml
	env $(KUBE_ENV) kubectl apply -f argocd/projects/sample-app.yaml
	env $(KUBE_ENV) kubectl apply -f argocd/applicationsets/env-bundle.yaml
```

- [ ] **Step 2: README の ArgoCD 手順を更新する**

`README.md` の ArgoCD 手順に以下の要点を入れる。

````markdown
ApplicationSet と生成される Application は `sample-applicationset` namespace に作成します。ArgoCD 本体は `argocd` namespace のままです。

`make install-argocd-apps` は、`argocd/namespaces/sample-applicationset.yaml`、`argocd/projects/sample-app.yaml`、`argocd/applicationsets/env-bundle.yaml` の順に適用します。このコマンドは ArgoCD 管理クラスタを向いた kubecontext で実行してください。

確認コマンド:

```bash
kubectl get applicationsets -n sample-applicationset
kubectl get applications -n sample-applicationset
kubectl describe application infra-core -n sample-applicationset
```
````

- [ ] **Step 3: TODO の確認コマンドを更新する**

`TODO.md` の該当コマンドを以下へ変更する。

```markdown
- [ ] `kubectl get applicationsets -n sample-applicationset`
- [ ] `kubectl describe applicationset env-bundle -n sample-applicationset`
- [ ] `kubectl get applications -n sample-applicationset`
- [ ] `kubectl describe application infra-core -n sample-applicationset`
- [ ] `kubectl describe application keda-operator -n sample-applicationset`
- [ ] `kubectl describe application sample-app -n sample-applicationset`
```

- [ ] **Step 4: README/TODO の参照更新を確認する**

Run:

```bash
rg -n "sample-applicationset|argocd/projects/sample-app.yaml|argocd/namespaces/sample-applicationset.yaml" README.md TODO.md Makefile
```

Expected: `README.md`、`TODO.md`、`Makefile` の3ファイルに `sample-applicationset` の参照が表示される。

- [ ] **Step 5: コミットする**

```bash
git add Makefile README.md TODO.md
git commit -m "docs: document applicationset namespace flow"
```

---

### Task 6: 最終検証を行う

**Files:**
- Test: `manifest/argocd/values.yaml`
- Test: `argocd/namespaces/sample-applicationset.yaml`
- Test: `argocd/projects/sample-app.yaml`
- Test: `argocd/applicationsets/env-bundle.yaml`
- Test: `sample-app/layout/layout_test.go`

- [ ] **Step 1: Argo CD chart render を確認する**

Run:

```bash
helm template argocd ./manifest/argocd -f manifest/argocd/values/develop.yaml -n argocd >/tmp/argocd-render.yaml
rg -n "application.namespaces|applicationsetcontroller.namespaces" /tmp/argocd-render.yaml
```

Expected: `application.namespaces` と `applicationsetcontroller.namespaces` がどちらも `sample-applicationset` を指す。

- [ ] **Step 2: ApplicationSet 関連 YAML の構文を確認する**

Run:

```bash
python3 -c 'import yaml; [yaml.safe_load(open(path)) for path in ["argocd/namespaces/sample-applicationset.yaml", "argocd/projects/sample-app.yaml", "argocd/applicationsets/env-bundle.yaml"]]; print("yaml ok")'
```

Expected: `yaml ok`

- [ ] **Step 3: レイアウト保護テストを実行する**

Run:

```bash
env GOPATH=$(pwd)/.cache/go GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./sample-app/layout
```

Expected: `ok`

- [ ] **Step 4: 全体テストを実行する**

Run:

```bash
make test
```

Expected: `go test ./...` が exit code 0 で完了する。

- [ ] **Step 5: Argo CD 導入済みクラスタで server dry-run を確認する**

Run:

```bash
kubectl apply --dry-run=server -f argocd/namespaces/sample-applicationset.yaml
kubectl apply --dry-run=server -f argocd/projects/sample-app.yaml
kubectl apply --dry-run=server -f argocd/applicationsets/env-bundle.yaml
```

Expected: 3コマンドすべてが server dry-run として成功する。CRD未導入環境では `AppProject` / `ApplicationSet` の resource mapping 解決に失敗するため、`make install-argocd` と `make argocd-ready` 後に実行する。

- [ ] **Step 6: 作業ツリーが整理されていることを確認する**

```bash
git status --short
```

Expected: Task 1からTask 5までのコミットが完了していれば出力なし。未コミットの差分がある場合は、該当タスクのコミット手順に戻って整理する。

---

## Self-Review

- `ApplicationSet` 本体と生成される `Application` の namespace を `sample-applicationset` に揃えるタスクを含めた。
- Argo CD 側で必要な `application.namespaces` と `applicationsetcontroller.namespaces` の有効化を含めた。
- Argo CD 側で ApplicationSet any namespace 用 RBAC を生成する `applicationSet.allowAnyNamespace` の有効化を含めた。
- `default` AppProject に user-controlled namespace を追加せず、専用 `sample-app` AppProject を作る方針にした。
- AppProject は `keda`、`default`、KEDA chart が使う `kube-system` を許可し、`CreateNamespace=true` 用の core `Namespace` と chart が render する resource kind に whitelist を限定した。
- `sample-applicationset` namespace 作成を Makefile の apply 導線に含め、namespace 不在で apply が失敗する問題を避けた。
- README/TODO/layout test の芋づる更新と、Helm render / YAML parse / Go test / server dry-run の検証を含めた。
