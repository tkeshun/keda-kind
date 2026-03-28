# keda-kind

kind 上で KEDA と ElasticMQ を使って SQS 風ワークロードをローカル検証するためのサンプルです。`enqueue` は一定間隔でメッセージを投入し、`dequeue` は KEDA `ScaledJob` で 1 件だけ処理して PostgreSQL に保存します。

## 構成

- `kind-config.yaml`: ingress-nginx を `localhost:8080` / `localhost:8443` で受けられる kind 設定
- `manifest/keda-operator`: KEDA upstream chart のラッパー
- `manifest/elasticmq`: ElasticMQ chart
- `manifest/postgresql`: PostgreSQL chart
- `manifest/enqueue-app`: 送信アプリ chart
- `manifest/dequeue-app`: `ScaledJob` ベースの受信アプリ chart
- `manifest/`: すべての Helm chart 配置先
- `sample-app/`: sample アプリ本体の配置先
- `compose.yaml`: KEDA なし検証環境

## 前提ツール

- Docker 29+
- kind 0.30+
- Helm 3.17+
- kubectl

## Go アプリのテスト

```bash
make test
```

## Docker イメージのビルド

```bash
make build
```

`docker buildx build` を使って `sample-app/docker/*.Dockerfile` から `local/enqueue:dev` と `local/dequeue:dev` をローカルに取り込みます。別アーキテクチャで試す場合は `PLATFORM=linux/arm64 make build` のように指定します。

## kind でのデプロイ

1. kind クラスタを作成します。

```bash
make kind-create
```

2. ingress-nginx を導入します。

```bash
make ingress
```

3. アプリイメージをビルドし、kind ノードへロードします。

```bash
make build
make kind-load
```

4. KEDA dependency を取得します。

```bash
make helm-deps
```

5. ミドルウェアと KEDA をインストールします。

```bash
make install-elasticmq
make install-postgresql
make install-keda
```

本番相当の AWS 権限で KEDA Operator を動かす場合は、KEDA Operator の ServiceAccount `keda-operator` に対して EKS Pod Identity Association を作成してから次を使います。

```bash
make install-keda-prod
```

6. アプリをデプロイします。

```bash
make install-enqueue
make install-dequeue
```

7. 動作確認を行います。

```bash
kubectl get pods
kubectl get scaledjobs
kubectl get jobs
kubectl logs deploy/enqueue
kubectl get deploy elasticmq postgresql
kubectl port-forward svc/postgresql 5432:5432
psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
```

`dequeue` はジョブとして起動するため、キューが空なら pod は残りません。`kubectl get jobs --watch` でジョブ生成を確認できます。KEDA は `keda` namespace に入るため、queue URL と ElasticMQ endpoint は chart 既定値で cluster FQDN を使います。kind の ingress 到達先は `http://127.0.0.1:8080` と `https://127.0.0.1:8443` です。

shared / app を含めた全 Helm chart は `manifest/` 配下にあります。app chart の開発用 override は `manifest/enqueue-app/values/develop.yaml` と `manifest/dequeue-app/values/develop.yaml` に置いています。

`make install-dequeue` は `manifest/dequeue-app/values/develop.yaml` を使い、KEDA scaler も `dequeue-config` Secret の static credential で ElasticMQ を参照します。kind / ElasticMQ 検証では追加の AWS identity 設定は不要です。

`manifest/keda-operator/values/production.yaml` は KEDA Operator 専用 ServiceAccount を作る前提です。production 向けの `dequeue` scaler は `identityOwner: operator` と `TriggerAuthentication.podIdentity.provider: aws-eks` を使うため、本番ではこの ServiceAccount に対して EKS Pod Identity Association を作成し、SQS 読み取り権限を付与します。

検証環境でも `make install-keda` は `keda-operator` ServiceAccount を作成します。ローカル検証では追加 annotation なし、本番では同名 ServiceAccount に対して EKS Pod Identity Association を作る想定です。

## docker compose での検証

1. ElasticMQ、PostgreSQL、enqueue を起動します。

```bash
make compose-up
```

2. 任意のタイミングで dequeue を 1 回実行します。

```bash
make compose-run-dequeue
```

3. PostgreSQL で保存結果を確認します。

```bash
docker compose exec postgresql psql -U app -d app -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
```

## 主要な環境変数

- `AWS_ENDPOINT`: ElasticMQ の SQS エンドポイント
- `AWS_REGION`: 既定値は `elasticmq`
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`: 既定値は `x`
- `QUEUE_NAME`: 既定値は `sample-queue`
- `SEND_INTERVAL`: enqueue の送信間隔。既定値は `5s`
- `QUEUE_URL`: dequeue が受信する完全な queue URL
- `DB_CONNECTION_STRING`: PostgreSQL 接続文字列

`enqueue` は送信前に SQS の `ApproximateNumberOfMessages` を参照し、可視メッセージ数が 10 件以上なら追加送信をスキップします。これは sample アプリ向けの best-effort な抑制であり、分散環境での厳密な上限制御を保証するものではありません。
