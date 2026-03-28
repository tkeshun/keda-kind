# TODO

## 実施した作業

- [x] SQS 風 sample アプリ本体を実装した
  - [x] `enqueue` が 5 桁コードと送信時刻を含む JSON を送信する
  - [x] `dequeue` が 1 件だけ処理して PostgreSQL に保存する
  - [x] PostgreSQL テーブルを自動作成する
- [x] sample アプリのコアロジックにテストを追加した
  - [x] `sample-app/internal/message`
  - [x] `sample-app/internal/enqueue`
  - [x] `sample-app/internal/dequeue`
  - [x] `sample-app/internal/config`
- [x] sample アプリ専用資産を `sample-app/` 配下へ移植した
  - [x] Go エントリポイントを `sample-app/cmd/` へ移した
  - [x] sample 専用 `internal` を `sample-app/internal/` へ移した
  - [x] Dockerfile を `sample-app/docker/` へ移した
  - [x] app Helm chart を `manifest/` へ集約した
  - [x] `Makefile` / `compose.yaml` / `README.md` の参照先を更新した
  - [x] 配置崩れを防ぐレイアウトテスト `sample-app/layout/layout_test.go` を追加した
- [x] 共有基盤 chart を `manifest/` 配下に整理した
  - [x] `manifest/keda-operator`
  - [x] `manifest/elasticmq`
  - [x] `manifest/postgresql`
- [x] Docker / compose / kind の基本導線を用意した
  - [x] buildx 前提のイメージビルド
  - [x] `compose.yaml` によるローカル検証構成
  - [x] `kind-config.yaml`
  - [x] `Makefile`
  - [x] `README.md`

## この環境で検証済み

- [x] `go test ./...`
- [x] `go test ./sample-app/layout`
- [x] `helm dependency build manifest/keda-operator`
- [x] `helm template enqueue ./manifest/enqueue-app`
- [x] `helm template dequeue ./manifest/dequeue-app`
- [x] `docker compose config`
- [x] `make build`

## 残作業
- 前提

export KUBECONFIG=/home/shun/dev/keda-kind/.cache/kubeconfig

- [ ] kind クラスタ上で `make kind-create` からデプロイ完走までを再確認する
  - [ ] `make kind-create`
    - [ ] `kubectl cluster-info`
    - [ ] `kubectl get nodes`
  - [ ] `make ingress`
    - [ ] `kubectl get pods -n ingress-nginx`
    - [ ] `kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=180s`
  - [ ] `make build`
    - [ ] `docker image inspect local/enqueue:dev`
    - [ ] `docker image inspect local/dequeue:dev`
  - [ ] `make kind-load`
    - [ ] `docker exec keda-kind-control-plane crictl images | grep 'local/enqueue'`
    - [ ] `docker exec keda-kind-control-plane crictl images | grep 'local/dequeue'`
  - [ ] `make helm-deps`
    - [ ] `test -f manifest/keda-operator/charts/keda-2.17.2.tgz`
  - [ ] `make install-elasticmq`
    - [ ] `helm status elasticmq`
    - [ ] `kubectl get deploy elasticmq`
    - [ ] `kubectl rollout status deployment/elasticmq --timeout=180s`
  - [ ] `make install-postgresql`
    - [ ] `helm status postgresql`
    - [ ] `kubectl get deploy postgresql`
    - [ ] `kubectl rollout status deployment/postgresql --timeout=180s`
  - [ ] `make install-keda` または `make install-keda-prod`
    - [ ] `helm status keda -n keda`
    - [ ] `kubectl get pods -n keda`
    - [ ] `kubectl rollout status deployment/keda-operator -n keda --timeout=180s`
  - [ ] `make install-enqueue`
    - [ ] `helm status enqueue`
    - [ ] `kubectl get deploy enqueue`
    - [ ] `kubectl rollout status deployment/enqueue --timeout=180s`
  - [ ] `make install-dequeue`
    - [ ] `helm status dequeue`
    - [ ] `kubectl get scaledjob dequeue`
    - [ ] `kubectl get triggerauthentication dequeue`
- [ ] kind 上で KEDA のスケーリング動作を確認する
  - [ ] `kubectl get scaledjobs`
  - [ ] `kubectl get jobs --watch`
  - [ ] `kubectl logs deploy/enqueue`
- [ ] kind 上の PostgreSQL に `dequeue` の保存結果が入ることを確認する
  - [ ] `kubectl get deploy postgresql`
  - [ ] `kubectl port-forward svc/postgresql 5432:5432`
  - [ ] `psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'`
- [ ] 必要なら chart の values を namespace / image tag / 実運用向け設定に合わせて調整する
- [ ] 必要なら CI 用の自動検証を追加する

## メモ

- sample アプリのコード正本は `sample-app/` 配下にある
- Helm chart の正本は shared / app ともに `manifest/` 配下にある
- 本ファイルの「検証済み」は、この環境で実行して通過したコマンドだけを記載している
