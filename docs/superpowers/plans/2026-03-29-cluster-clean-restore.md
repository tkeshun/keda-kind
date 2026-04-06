# Cluster Clean/Restore Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `ingress-nginx` を残したまま app/DB/KEDA の Helm release 一式を削除し、既存 install target を再利用して復帰できる Makefile と手順書を追加する

**Architecture:** Makefile に `cluster-clean` と `cluster-restore` を追加し、cleanup は Helm uninstall 群、restore は既存 `install-*` target 依存に寄せる。README と TODO に操作手順と前提条件を追記し、検証は live cluster 上で Helm release の消失と復帰、`ingress-nginx` の存続を確認する。

**Tech Stack:** GNU Make, Helm, kubectl, kind, Markdown

---

### Task 1: Makefile に cluster clean/restore を追加する

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: `Makefile` の target 定義位置を確認する**

Run:

```bash
sed -n '1,120p' Makefile
```

Expected: `.PHONY`、`install-*`、`enqueue-scale-*` が並んでいて、新 target を同じまとまりに追加できる。

- [ ] **Step 2: `.PHONY` に新 target を追加する**

`Makefile` の `.PHONY` 行を次の形に更新する。

```make
.PHONY: test build-enqueue build-dequeue build kind-create kind-load ingress helm-deps install-elasticmq install-postgresql install-keda install-keda-prod install-enqueue install-dequeue enqueue-scale-zero enqueue-scale-one cluster-clean cluster-restore compose-up compose-run-dequeue
```

- [ ] **Step 3: `cluster-clean` を実装する**

`install-dequeue` の直後に次を追加する。

```make
cluster-clean:
	-env $(KUBE_ENV) $(HELM_ENV) helm uninstall dequeue
	-env $(KUBE_ENV) $(HELM_ENV) helm uninstall enqueue
	-env $(KUBE_ENV) $(HELM_ENV) helm uninstall postgresql
	-env $(KUBE_ENV) $(HELM_ENV) helm uninstall elasticmq
	-env $(KUBE_ENV) $(HELM_ENV) helm uninstall keda -n keda
```

ポイント:

- 各行の先頭 `-` で release 不在時エラーを吸収する
- すべて `$(KUBE_ENV)` と `$(HELM_ENV)` を通す
- `keda` だけ namespace 指定を入れる

- [ ] **Step 4: `cluster-restore` を実装する**

`cluster-clean` の直後に次を追加する。

```make
cluster-restore: install-elasticmq install-postgresql install-keda install-enqueue install-dequeue
```

ポイント:

- install ロジックは複製しない
- `build`、`kind-load`、`helm-deps` は含めない

- [ ] **Step 5: `make -n` で生成コマンドを確認する**

Run:

```bash
make -n cluster-clean
make -n cluster-restore
```

Expected:

- `cluster-clean` は 5 個の `helm uninstall` を表示する
- `cluster-restore` は既存の `install-*` target 展開を表示する

- [ ] **Step 6: 変更を commit する**

Run:

```bash
git add Makefile
git commit -m "feat: add cluster clean and restore targets"
```

Expected: Makefile 変更だけを含む commit が作られる。

### Task 2: README に clean/restore 運用導線を追加する

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 既存の install 手順まわりを確認する**

Run:

```bash
sed -n '36,130p' README.md
```

Expected: `make kind-create` から `make install-dequeue`、`enqueue-scale-*` 説明までが読める。

- [ ] **Step 2: clean/restore 説明を追記する**

`enqueue-scale-*` の説明の近くに、次の趣旨の段落を追加する。

```md
app/DB/KEDA release をまとめて消したい場合は `make cluster-clean` を使います。`ingress-nginx` と kind cluster 自体は残ります。

戻すときは、事前に `make build`、`make kind-load`、`make helm-deps` を済ませたうえで `make cluster-restore` を実行します。
```

要件:

- `ingress-nginx` を残すことを明記する
- restore の前提条件を 1 文で明記する
- 既存の install フローを壊さない位置に置く

- [ ] **Step 3: README の差分を確認する**

Run:

```bash
git diff -- README.md
```

Expected: clean/restore の説明だけが追記されている。

- [ ] **Step 4: 変更を commit する**

Run:

```bash
git add README.md
git commit -m "docs: describe cluster clean and restore workflow"
```

Expected: README 変更だけを含む commit が作られる。

### Task 3: TODO に clean/restore メモを追加する

**Files:**
- Modify: `TODO.md`

- [ ] **Step 1: TODO の実行手順セクションを確認する**

Run:

```bash
sed -n '40,120p' TODO.md
```

Expected: `make kind-create`、`make ingress`、`make build` 以降のチェック項目が見える。

- [ ] **Step 2: clean/restore の運用メモを追記する**

`make install-dequeue` や検証項目の近くに、次のようなメモを追加する。

```md
- [ ] app/DB/KEDA release をまとめて掃除したいときは `make cluster-clean` を使う
- [ ] 復帰するときは `make build`、`make kind-load`、`make helm-deps` 済みで `make cluster-restore` を使う
```

要件:

- user 要求どおり `TODO.md` に残す
- 既存 checklist の文体に合わせる
- 完了済み扱いにはせず、補助メモとして残す

- [ ] **Step 3: TODO の差分を確認する**

Run:

```bash
git diff -- TODO.md
```

Expected: clean/restore 用の 2 行だけが自然な位置に追加されている。

- [ ] **Step 4: 変更を commit する**

Run:

```bash
git add TODO.md
git commit -m "docs: add cluster clean restore notes to todo"
```

Expected: TODO 変更だけを含む commit が作られる。

### Task 4: live cluster で clean/restore を検証する

**Files:**
- Modify: none

- [ ] **Step 1: 現在の Helm release 一覧を確認する**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm list -A
```

Expected: `elasticmq`、`postgresql`、`keda`、`enqueue`、`dequeue` が見える。

- [ ] **Step 2: clean を実行する**

Run:

```bash
make cluster-clean
```

Expected: 各 `helm uninstall` が走り、release 不在があっても make 全体は継続する。

- [ ] **Step 3: cleanup 結果を確認する**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm list -A
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get pods -n ingress-nginx
```

Expected:

- `helm list -A` に対象 5 release が出ない
- `ingress-nginx-controller` pod は残っている

- [ ] **Step 4: restore を実行する**

Run:

```bash
make cluster-restore
```

Expected: `install-elasticmq`、`install-postgresql`、`install-keda`、`install-enqueue`、`install-dequeue` が順に走る。

- [ ] **Step 5: 復帰結果を確認する**

Run:

```bash
env KUBECONFIG=$(pwd)/.cache/kubeconfig HELM_CONFIG_HOME=$(pwd)/.cache/helm/config HELM_CACHE_HOME=$(pwd)/.cache/helm/cache HELM_DATA_HOME=$(pwd)/.cache/helm/data helm list -A
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl rollout status deployment/elasticmq --timeout=180s
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl rollout status deployment/postgresql --timeout=180s
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl rollout status deployment/keda-operator -n keda --timeout=180s
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl rollout status deployment/enqueue --timeout=180s
env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get scaledjob dequeue
```

Expected:

- `helm list -A` に対象 5 release が戻る
- 各 deployment rollout が成功する
- `scaledjob dequeue` が存在する

- [ ] **Step 6: 検証結果を commit する**

Run:

```bash
git commit --allow-empty -m "test: verify cluster clean restore workflow"
```

Expected: 実ファイル変更がなくても、検証済み記録の commit を残せる。

