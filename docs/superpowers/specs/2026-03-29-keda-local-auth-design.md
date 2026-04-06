# KEDA Local Auth Switch Design

## Goal

kind 上の local 検証では KEDA の SQS scaler が static credential で ElasticMQ を読めるようにしつつ、production では従来どおり KEDA operator の AWS identity を使う。

## Current State

- `manifest/dequeue-app/templates/scaledjob.yaml` は `identityOwner: operator` を固定で出力している
- `manifest/dequeue-app/templates/triggerauthentication.yaml` は `podIdentity.provider: aws-eks` を固定で出力している
- `make install-dequeue` は `manifest/dequeue-app/values/develop.yaml` を読み込むが、develop values に KEDA 認証方式の切り替えがない
- kind 上では KEDA operator が IMDS から認証情報を取得できず、`ScaledJob` が `PartialTriggerError` で止まる

## Design

### Authentication Modes

`dequeue-app` chart に KEDA scaler 向け認証モードを追加し、次の 2 方式だけをサポートする。

1. `secret`
   - local / ElasticMQ 用
   - KEDA scaler が `TriggerAuthentication.secretTargetRef` 経由で `AWS_ACCESS_KEY_ID` と `AWS_SECRET_ACCESS_KEY` を読む
   - `identityOwner` は出力しない
2. `operator`
   - production / AWS 用
   - KEDA scaler が `identityOwner: operator` を使う
   - `TriggerAuthentication.podIdentity.provider: aws-eks` を出力する

自動判定は入れない。認証方式は values で明示的に選ぶ。

### Values Shape

`manifest/dequeue-app/values.yaml` に KEDA 認証専用の設定を追加する。

```yaml
kedaAuthentication:
  mode: operator
  podIdentityProvider: aws-eks
```

`manifest/dequeue-app/values/develop.yaml` では local 用に次を上書きする。

```yaml
image:
  repository: local/dequeue
  tag: dev

kedaAuthentication:
  mode: secret
```

`values.yaml` の既定値は production 寄りのまま維持する。`make install-dequeue` はすでに develop override を読むため、local 検証では追加の CLI 引数なしで `secret` mode が有効になる。

### Template Changes

`manifest/dequeue-app/templates/scaledjob.yaml`

- `aws-sqs-queue` trigger の `identityOwner` は `kedaAuthentication.mode == "operator"` のときだけ出力する
- `authenticationRef` は既存どおり `dequeue` を参照する

`manifest/dequeue-app/templates/triggerauthentication.yaml`

- `kedaAuthentication.mode == "secret"` のとき:
  - `secretTargetRef` を出力する
  - `AWS_ACCESS_KEY_ID` と `AWS_SECRET_ACCESS_KEY` を `dequeue-config` Secret から参照する
- `kedaAuthentication.mode == "operator"` のとき:
  - `podIdentity.provider` を出力する
- 想定外の mode は template failure にしてよい。silent fallback はしない

`dequeue` Job 本体の env は変更しない。アプリ本体が queue を読むための AWS credential と、KEDA scaler が queue length を読むための credential を同じ Secret で共有する。

## Documentation Changes

- `README.md` に local / production の認証差分を追記する
- `TODO.md` の検証手順メモを、develop は static credential で KEDA を動かす前提に合わせる

## Verification

- `helm template dequeue ./manifest/dequeue-app`
- `helm template dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml`
- kind 上で `make install-dequeue`
- `kubectl describe scaledjob dequeue` で `KEDAScalerFailed` が消えること
- `kubectl get jobs --watch` で job が生成されること
- PostgreSQL に `dequeue` の保存結果が入ること

## Non-Goals

- `enqueue` の送信ロジック変更
- KEDA operator chart 自体の認証方式変更
- production の AWS identity 運用変更
