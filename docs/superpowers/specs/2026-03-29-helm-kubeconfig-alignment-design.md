# Helm Kubeconfig Alignment Design

## Goal

repo の Makefile から実行する Helm 操作が常に `.cache/kubeconfig` を使うようにし、`kind-keda-kind` 以外の current context を誤って更新しないようにする。

## Current State

- `Makefile` では `kind` と `kubectl` 向けに `KUBE_ENV = KUBECONFIG=$(KUBECONFIG_PATH)` を定義している
- ただし Helm 実行系 target は `env $(HELM_ENV) helm ...` だけで、`KUBECONFIG` を渡していない
- そのため `make install-dequeue` は shell の default context を更新しうる
- 実際に `helm get manifest dequeue` は新しい develop 用 manifest を返す一方、`.cache/kubeconfig` 側の live resource は古い `podIdentity: aws-eks` のままだった

## Design

### Makefile Alignment

Helm を呼ぶ target はすべて `env $(KUBE_ENV) $(HELM_ENV) helm ...` に統一する。

対象:

- `helm-deps`
- `install-elasticmq`
- `install-postgresql`
- `install-keda`
- `install-keda-prod`
- `install-enqueue`
- `install-dequeue`

これにより repo の主要導線は、kind 作成、kubectl 検証、Helm install / upgrade が同一の kubeconfig を共有する。

### Documentation

- `README.md` に、Make target は `.cache/kubeconfig` を使う前提で cluster を操作することを追記する
- `TODO.md` の前提 `export KUBECONFIG=...` は補助情報に下げ、Makefile 自体が kubeconfig を固定することを明記する
- 再検証項目に「live TriggerAuthentication が `secretTargetRef` になっていること」を加える

## Verification

- `make install-dequeue`
- `env KUBECONFIG=$(pwd)/.cache/kubeconfig helm list -A`
- `env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get triggerauthentication dequeue -o yaml`
- `env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get scaledjob dequeue -o yaml`
- `env KUBECONFIG=$(pwd)/.cache/kubeconfig kubectl get jobs --watch`
- PostgreSQL への保存確認

## Non-Goals

- Docker build / compose target の変更
- kind cluster 作成ロジックの変更
- KEDA auth template 自体の再設計
