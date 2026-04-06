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

`export KUBECONFIG=/home/shun/dev/keda-kind/.cache/kubeconfig` は、このチェックリスト内で手で叩く `kubectl` / `helm` コマンドの shell 上の便宜であり、Makefile の正しさには不要。

- [ ] kind クラスタ上で `make kind-create` からデプロイ完走までを再確認する
  - [ ] `make kind-create`
    - [ ] `kubectl cluster-info`
    - [ ] `kubectl get nodes`
  - [ ] `make ingress`
    - [ ] `kubectl get pods -n ingress-nginx`
    - [ ] `kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=180s`
  - [x] `make build`
    - [x] `docker image inspect local/enqueue:dev`
    - [x] `docker image inspect local/dequeue:dev`
  - [x] `make kind-load`
    - [x] `docker exec keda-kind-control-plane crictl images | grep 'local/enqueue'`
    - [x] `docker exec keda-kind-control-plane crictl images | grep 'local/dequeue'`
  - [x] `make helm-deps`
    - [x] `test -f manifest/keda-operator/charts/keda-2.18.1.tgz`
  - [x] `make install-elasticmq`
    - [x] `helm status elasticmq`
    - [x] `kubectl get deploy elasticmq`
    - [x] `kubectl rollout status deployment/elasticmq --timeout=180s`
  - [x] `make install-postgresql`
    - [x] `helm status postgresql`
    - [x] `kubectl get deploy postgresql`
    - [x] `kubectl rollout status deployment/postgresql --timeout=180s`
  - [x] `make install-keda` または `make install-keda-prod`
    - [x] `helm status keda -n keda`
    - [x] `kubectl get pods -n keda`
    - [x] `kubectl rollout status deployment/keda-operator -n keda --timeout=180s`
  - [x] `make install-enqueue`
    - [x] `helm status enqueue`
    - [x] `kubectl get deploy enqueue`
    - [x] `kubectl rollout status deployment/enqueue --timeout=180s`
    - [ ] scheduled モード確認
      - [ ] `kubectl get deploy enqueue -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ENQUEUE_MODE")].value}'`
      - [ ] `kubectl logs deploy/enqueue`
  - [ ] `make install-enqueue-http`
    - [ ] `helm status enqueue`
    - [ ] `kubectl get deploy enqueue`
    - [ ] `kubectl rollout status deployment/enqueue --timeout=180s`
    - [ ] HTTP モード確認
      - [ ] `kubectl get deploy enqueue -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ENQUEUE_MODE")].value}'`
      - [ ] `kubectl logs deploy/enqueue`
    - [ ] 手動リクエストを投げる
      - [ ] `kubectl port-forward svc/enqueue 18080:8080`
      - [ ] `curl -i http://127.0.0.1:18080/healthz`
      - [ ] `curl -i -X POST http://127.0.0.1:18080/enqueue`
  - [x] `make install-dequeue`
    - [x] `helm status dequeue`
    - [x] `kubectl get scaledjob dequeue`
    - [x] `kubectl get triggerauthentication dequeue`
    - [x] `kubectl get triggerauthentication dequeue -o yaml`
      - [x] `secretTargetRef` が live resource に入っている
- [x] kind 上で KEDA のスケーリング動作を確認する
  - [x] `kubectl get scaledjobs`
  - [x] `kubectl get jobs --watch`
  - [x] `kubectl logs deploy/enqueue`
- [x] kind 上の PostgreSQL に `dequeue` の保存結果が入ることを確認する
  - [x] `kubectl get deploy postgresql`
  - [x] `kubectl port-forward svc/postgresql 5432:5432`
  - [x] `psql 'postgres://app:app@127.0.0.1:5432/app?sslmode=disable' -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'`
  - [x] `kubectl exec deploy/postgresql -- psql -U app -d app -c 'select code, sent_at, stored_at from queue_messages order by id desc limit 10;'`
- [ ] PostgreSQL の残存データを消して日本時間運用を確認する
  - [ ] `kubectl exec deploy/postgresql -- psql -U app -d app -c "delete from queue_messages;"`
  - [ ] `kubectl exec deploy/postgresql -- psql -U app -d app -c "ALTER SYSTEM SET timezone = 'Asia/Tokyo';"`
  - [ ] `kubectl exec deploy/postgresql -- psql -U app -d app -c "select pg_reload_conf();"`
  - [ ] `kubectl exec deploy/postgresql -- psql -U app -d app -c "show timezone;"`
- [ ] 必要なら chart の values を namespace / image tag / 実運用向け設定に合わせて調整する
- [ ] 必要なら CI 用の自動検証を追加する

## 追加タスク

- [ ] ArgoCDで管理できるようにする
- [ ] queue数に応じたスケールを再現する
  - [ ] k6でシナリオ書く
- [ ] requestが足らないときの挙動を再現できるようにする
  - [ ] 過剰なリクエストのJob定義を用意する
  - [ ] K6でシナリオを書く
- [ ] 
## メモ

- sample アプリのコード正本は `sample-app/` 配下にある
- Helm chart の正本は shared / app ともに `manifest/` 配下にある
- sample アプリの SQS client は AWS SDK default credential chain を使う
- local ElasticMQ は chart の Secret が `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` を pod に入れるので動く
- production は app / scaler ともに code branch なしで Pod Identity へ切り替える想定
- production の `dequeue` scaler は `identityOwner: operator` と `aws-eks` Pod Identity を使う
- production では KEDA Operator と app の対象 ServiceAccount に EKS Pod Identity Association を付ける
- 本ファイルの「検証済み」は、この環境で実行して通過したコマンドだけを記載している
