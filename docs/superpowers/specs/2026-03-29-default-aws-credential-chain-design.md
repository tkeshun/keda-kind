# Default AWS Credential Chain Design

## Goal

`enqueue` と `dequeue` の SQS client が AWS SDK v2 の default credential chain を使うようにし、local では環境変数、production では EKS Pod Identity から同じコードで認証できるようにする。

## Current State

- `sample-app/internal/config/config.go` は `AWS_ACCESS_KEY_ID` と `AWS_SECRET_ACCESS_KEY` を `config.AWS` に取り込んでいる
- `sample-app/internal/adapters/sqs/client.go` は `credentials.NewStaticCredentialsProvider(...)` を明示している
- そのため env に値があれば常に static credential を使い、Pod Identity などの default chain を利用できない

## Design

### Config Responsibility

`config.AWS` は接続先解決に必要な値だけを持つ。

残す:

- `Endpoint`
- `Region`

削除する:

- `AccessKeyID`
- `SecretAccessKey`

`loadAWS` は `AWS_ENDPOINT` を必須、`AWS_REGION` を任意のまま維持する。credential 系 env は config レイヤでは読まない。

### SQS Client Responsibility

`sample-app/internal/adapters/sqs/client.go` の `New` は次の形に寄せる。

- `awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))`
- `BaseEndpoint` は引き続き `cfg.Endpoint` を設定する
- `WithCredentialsProvider(...)` は使わない

これにより認証ソースは SDK に委譲される。

- local: `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` を env から取得
- production: Pod Identity / IAM Role などの default chain から取得
- ElasticMQ: env にダミー値 `x` を入れておけば従来どおり通る

### Manifest / Docs Impact

- app chart の Secret と env 配線は local のために残してよい
- `dequeue-app` の `TriggerAuthentication.secretTargetRef` も local では維持する
- README / TODO では「アプリ本体は default credential chain を使う」ことを説明し、local は env、production は Pod Identity で同じコードが動くと書く

## Verification

- `go test ./...`
- `helm template enqueue ./manifest/enqueue-app`
- `helm template dequeue ./manifest/dequeue-app`
- `helm template dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml`
- kind 上で `make install-enqueue`
- kind 上で `make install-dequeue`
- `kubectl get jobs --watch`
- PostgreSQL に保存結果が入ること

## Non-Goals

- queue payload や dequeue ロジックの変更
- KEDA scaler の認証方式変更
- endpoint / region の扱い変更
