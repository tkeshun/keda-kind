# ArgoCD デプロイ管理導入設計

## 1. 目的

本設計は、`keda-kind` リポジトリに ArgoCD を導入し、既存の Helm chart 群を GitOps で管理できるようにするための初期構成を定義する。

初回導入では次を優先する。

- `kind` 1 クラスタで成立すること
- ArgoCD 本体をこのリポジトリで管理できること
- 基盤 chart と sample app chart を分離して同期できること
- 後続で `ApplicationSet` に拡張しやすいこと

## 2. 採用方針

- ArgoCD 本体は `manifest/argocd` の wrapper chart で管理する
- 初回は `ApplicationSet` ではなく静的 `Application` を使う
- 管理対象は `infra-core`、`keda-operator`、`sample-app` の 3 Application に分ける
- 既存 chart は直接 Application から複数参照せず、bundle chart で束ねる
- `argocd` CLI や関連ツールのダウンロードが必要な場合は自動取得せず、ユーザーに理由付きで案内する

## 3. 構成

### 3.1 ArgoCD 本体

- `manifest/argocd`
  - upstream `argo/argo-cd` chart の wrapper chart
  - `argocd` namespace にインストールする
  - 初回は kind 向けの最小設定のみ持つ

### 3.2 管理対象 Application

- `argocd/applications/infra-core.yaml`
  - `manifest/infra-bundle` を参照する
  - `elasticmq`, `postgresql` を束ねる
- `argocd/applications/keda-operator.yaml`
  - `manifest/keda-operator` を直接参照する
  - `keda` namespace にデプロイする
- `argocd/applications/sample-app.yaml`
  - `manifest/app-bundle` を参照する
  - `enqueue-app`, `dequeue-app` を束ねる

### 3.3 bundle chart

- `manifest/infra-bundle`
  - infra 系 chart の dependency を持つ
- `manifest/app-bundle`
  - app 系 chart の dependency を持つ

## 4. 運用手順

初回の導入順は以下とする。

1. `make kind-create`
2. `make helm-deps`
3. `make helm-deps-argocd`
4. `make install-argocd`
5. `make argocd-ready`
6. `kubectl apply -f argocd/applications/infra-core.yaml`
7. `kubectl apply -f argocd/applications/keda-operator.yaml`
8. `kubectl apply -f argocd/applications/sample-app.yaml`

## 5. 制約

- local image は引き続き `make build` と `make kind-load` で kind ノードへ投入する
- ArgoCD 導入後も、ローカル検証では image registry の自動配布は行わない
- `argocd` CLI を前提にしない
- `keda-operator` は既存 chart の namespace 前提を保つため、初回は bundle に含めず単独 Application に分離する
- 将来の `ApplicationSet + Cluster generator` への拡張余地を残すため、ArgoCD 制御定義は `argocd/` 配下へ分離する

## 6. 将来拡張

- `argocd/applicationsets/` を追加し、静的 `Application` から移行する
- cluster Secret ラベルを使った環境別配布に広げる
- `develop` / `production` values の切替を `ApplicationSet` 側で管理する
