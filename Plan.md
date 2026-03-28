# KEDA を kind で検証する詳細計画

## 目標

kind クラスタ上で KEDA による SQS 風スケーリングをローカル再現する。SQS 代替として同一クラスタ内に ElasticMQ を配置し、`enqueue` がキューへ投入したメッセージを `dequeue` が `ScaledJob` で 1 件ずつ処理して PostgreSQL に保存できる状態を完成条件とする。

あわせて、KEDA を使わないローカル疎通確認経路として docker compose 環境も用意し、キュー投入から DB 保存までを Kubernetes 外でも再現できるようにする。

## 最終成果物

### 1. kind 基盤

- `kind-config.yaml`
  - kind 単一ノードクラスタ定義
  - ingress-nginx 用の `extraPortMappings`
  - `ingress-ready=true` ノードラベル

### 2. Helm Chart

- `manifest/keda-operator`
  - upstream `kedacore/keda` chart を依存としてラップする chart
  - kind 用の軽量設定を values で上書きする
- `manifest/elasticmq`
  - `Deployment`
  - `Service`
  - 起動時 queue 作成を含む `ConfigMap`
- `manifest/postgresql`
  - `Deployment`
  - `Service`
  - 認証情報 `Secret`
- `manifest/enqueue-app`
  - `Deployment`
  - AWS 認証情報 `Secret`
- `manifest/dequeue-app`
  - `ScaledJob`
  - `TriggerAuthentication`
  - 接続情報 `Secret`

### 3. Go アプリケーション

- `enqueue`
  - 一定間隔で ElasticMQ にメッセージ投入
  - メッセージは 5 桁ランダム値と送信時刻を含む JSON
  - 送信間隔や接続先は環境変数化
- `dequeue`
  - Go 実装
  - 1 メッセージだけ受信して PostgreSQL に保存
  - 保存後はメッセージを削除して終了

### 4. ローカル検証環境

- `compose.yaml`
  - ElasticMQ
  - PostgreSQL
  - enqueue
  - 手動起動用 dequeue

### 5. 手順書

- `README.md`
  - buildx によるイメージビルド手順
  - kind への image load 手順
  - Helm install 順序
  - compose 検証手順
  - 動作確認コマンド

## 実装フェーズ

### フェーズ 1: kind と ingress

1. kind 設定ファイルを作成する
2. ingress-nginx を kind 向け構成で導入する
3. localhost から ingress へ到達できる状態を作る

### フェーズ 2: ミドルウェア

1. KEDA ラッパー chart を作る
2. ElasticMQ chart を作る
3. PostgreSQL chart を作る
4. 依存順に install できるようにする

### フェーズ 3: サンプルアプリ

1. `enqueue` を Go で作る
2. `dequeue` を Go で作る
3. `dequeue` が PostgreSQL の保存先テーブルを自動作成できるようにする
4. Dockerfile を buildx 前提で作る

### フェーズ 4: Kubernetes デプロイ

1. `enqueue` 用 chart を作る
2. `dequeue` 用 chart を作る
3. `dequeue` は `ScaledJob` で起動するようにする
4. KEDA の SQS scaler が ElasticMQ を参照できるようにする

### フェーズ 5: compose 検証と手順整理

1. KEDA なしで確認できる compose 構成を作る
2. kind へのコンテナイメージ登録手順を整理する
3. アプリのデプロイ手順を整理する
4. DB 保存確認まで含めた検証コマンドを文書化する

## インターフェース定義

### キューメッセージ形式

```json
{
  "code": "12345",
  "sent_at": "2026-03-29T10:00:00Z"
}
```

### 主要環境変数

#### enqueue

- `AWS_ENDPOINT`
- `AWS_REGION`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `QUEUE_NAME`
- `SEND_INTERVAL`

#### dequeue

- `AWS_ENDPOINT`
- `AWS_REGION`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `QUEUE_NAME`
- `QUEUE_URL`
- `DB_CONNECTION_STRING`

## 完了条件

- `go test ./...` が通る
- `docker buildx build` で enqueue/dequeue イメージをローカル build できる
- `helm template` で全 chart が展開できる
- kind 上で ElasticMQ、PostgreSQL、KEDA、アプリを順に導入できる
- `enqueue` の投入に応じて `dequeue` の job が起動する
- PostgreSQL に処理結果が保存される
- docker compose でも同等の疎通確認ができる

## 参考情報

- KEDA chart: https://github.com/kedacore/charts
- KEDA ScaledJob docs: https://keda.sh/docs/2.19/reference/scaledjob-spec/
- KEDA AWS SQS scaler docs: https://keda.sh/docs/2.4/scalers/aws-sqs/
- ElasticMQ: https://github.com/softwaremill/elasticmq
