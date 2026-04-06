# keda-kind

kind 上で KEDA と ElasticMQ を使って SQS 風ワークロードをローカル検証するためのサンプルです。`enqueue` は一定間隔でメッセージを投入し、`dequeue` は KEDA `ScaledJob` で 1 件だけ処理して PostgreSQL に保存します。

## 構成

- `kind-config.yaml`: ingress-nginx を `localhost:8080` / `localhost:8443` で受けられる kind 設定
- `manifest/keda-operator`: KEDA upstream chart のラッパー
- `manifest/argocd`: ArgoCD upstream chart のラッパー
- `manifest/elasticmq`: ElasticMQ chart
- `manifest/postgresql`: PostgreSQL chart
- `manifest/enqueue-app`: 送信アプリ chart
- `manifest/dequeue-app`: `ScaledJob` ベースの受信アプリ chart
- `manifest/infra-bundle`: ElasticMQ / PostgreSQL を束ねる bundle chart
- `manifest/app-bundle`: enqueue / dequeue を束ねる bundle chart
- `argocd/applications/`: ArgoCD Application 定義
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

repo の `make` target は `KUBECONFIG=$(CURDIR)/.cache/kubeconfig` を使う前提で Helm / kubectl / kind を実行します。default context が別クラスタを向いていても、repo の導線は `kind-keda-kind` を対象にします。

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

ArgoCD chart dependency を取得する場合は次を使います。

```bash
make helm-deps-argocd
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

`enqueue` の deployment を一時停止したい場合は `make enqueue-scale-zero`、再開したい場合は `make enqueue-scale-one` を使います。

HTTP モードで `enqueue` をデプロイしたい場合は、次を使います。

```bash
make install-enqueue-http
```

`make cluster-clean` は app / DB / KEDA の release だけを削除し、ingress-nginx と kind クラスタ本体、ロード済みイメージは残します。`make cluster-restore` は `make build`、`make kind-load`、`make helm-deps` を先に終えた local/develop の状態を前提に、`install-keda` と app の develop values を戻します。

7. 動作確認を行います。

```bash
kubectl get pods
kubectl get scaledjobs
kubectl get jobs
kubectl logs deploy/enqueue
kubectl get deploy elasticmq postgresql
kubectl port-forward svc/postgresql 5432:5432
psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
kubectl exec deploy/postgresql -- psql -U app -d app -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'
```

`dequeue` はジョブとして起動するため、キューが空なら pod は残りません。`kubectl get jobs --watch` でジョブ生成を確認できます。KEDA は `keda` namespace に入るため、queue URL と ElasticMQ endpoint は chart 既定値で cluster FQDN を使います。kind の ingress 到達先は `http://127.0.0.1:8080` と `https://127.0.0.1:8443` です。

shared / app を含めた全 Helm chart は `manifest/` 配下にあります。app chart の開発用 override は `manifest/enqueue-app/values/develop.yaml` と `manifest/dequeue-app/values/develop.yaml` に置いています。

sample アプリの SQS client は AWS SDK の default credential chain を使います。kind / ElasticMQ では chart の Secret が `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` を pod に入れるため、追加の AWS identity 設定は不要です。

`make install-dequeue` は `manifest/dequeue-app/values/develop.yaml` を使い、KEDA scaler も同じ Secret 経由の認証情報で ElasticMQ を参照します。production では app / scaler ともに code branch なしで Pod Identity に切り替えられます。

`manifest/keda-operator/values/production.yaml` は KEDA Operator 専用 ServiceAccount を作る前提です。production 向けの `dequeue` scaler は `identityOwner: operator` と `TriggerAuthentication.podIdentity.provider: aws-eks` を使うため、本番ではこの ServiceAccount に対して EKS Pod Identity Association を作成し、SQS 読み取り権限を付与します。app 側も同様に、対象 ServiceAccount に Pod Identity を関連付ければ AWS SDK default credential chain で認証されます。

検証環境でも `make install-keda` は `keda-operator` ServiceAccount を作成します。ローカル検証では追加 annotation なし、本番では同名 ServiceAccount に対して EKS Pod Identity Association を作る想定です。

## ArgoCD でのデプロイ管理

1. ArgoCD chart dependency を取得します。

```bash
make helm-deps-argocd
```

2. ArgoCD を導入します。

```bash
make install-argocd
make argocd-ready
```

3. ArgoCD 用の初期 admin password を確認し、必要なら port-forward で UI に入ります。

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d; echo
kubectl -n argocd port-forward svc/argocd-server 8081:80
```

4. `argocd/applications/infra-core.yaml`、`argocd/applications/keda-operator.yaml`、`argocd/applications/sample-app.yaml` の `repoURL` を、ArgoCD から到達できる Git remote に置き換えます。

このリポジトリには現時点で Git remote が設定されていないため、Application 定義の `repoURL` はプレースホルダーです。置き換え前に apply すると ArgoCD は同期できません。

5. Application を適用します。

```bash
make install-argocd-apps
```

ArgoCD CLI はこの導線では必須ではありません。CLI のダウンロードや追加ツールの導入が必要な場合は、自動取得せずユーザーが実施する前提です。

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
- `ENQUEUE_MODE`: `enqueue` の動作モード。`scheduled` または `http`。既定値は `scheduled`
- `SEND_INTERVAL`: enqueue の送信間隔。既定値は `5s`
- `HTTP_PORT`: `ENQUEUE_MODE=http` のときの listen port。既定値は `8080`
- `QUEUE_URL`: dequeue が受信する完全な queue URL
- `DB_CONNECTION_STRING`: PostgreSQL 接続文字列

`enqueue` は送信前に SQS の `ApproximateNumberOfMessages` を参照し、可視メッセージ数が 10 件以上なら追加送信をスキップします。これは sample アプリ向けの best-effort な抑制であり、分散環境での厳密な上限制御を保証するものではありません。

`ENQUEUE_MODE=http` にすると `enqueue` は定期投入を止め、HTTP サーバーとして起動します。`POST /enqueue` で 1 件投入を試み、`GET /healthz` で疎通確認できます。

```bash
docker compose run --rm -p 8080:8080 \
  -e ENQUEUE_MODE=http \
  enqueue

curl -i -X POST http://127.0.0.1:8080/enqueue
curl -i http://127.0.0.1:8080/healthz
```
