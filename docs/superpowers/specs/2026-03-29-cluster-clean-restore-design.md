# Cluster Clean/Restore Design

## Summary

kind クラスタ上の `ingress-nginx` を残したまま、アプリ系 Helm release 一式をきれいに削除し、既存 install target を使って復帰できる Makefile 導線を追加する。

対象 release は次の 5 つ:

- `elasticmq`
- `postgresql`
- `keda`
- `enqueue`
- `dequeue`

`cluster-clean` はこれらを Helm uninstall で削除する。`cluster-restore` は既存の `install-*` target を再利用して復帰する。`build`、`kind-load`、`helm-deps` は復帰 target に含めず、事前条件として扱う。

## Goals

- `ingress-nginx` を残したままアプリ系 release だけをまとめて削除できる
- 既存の install 手順を再利用して安全に復帰できる
- release 不在時でも `cluster-clean` が止まらない
- README と TODO に運用手順を残し、手元確認の導線を揃える

## Non-Goals

- kind cluster 自体の削除
- `ingress-nginx` の削除や再作成
- `make build`、`make kind-load`、`make helm-deps` の自動実行
- 個別 release ごとの細かな停止制御の追加

## Makefile Changes

### `cluster-clean`

`cluster-clean` を [Makefile](/home/shun/dev/keda-kind/Makefile) に追加する。

動作:

- `dequeue`
- `enqueue`
- `postgresql`
- `elasticmq`
- `keda` (`-n keda`)

を順に `helm uninstall` する。

要件:

- すべて `env $(KUBE_ENV) $(HELM_ENV) helm ...` で実行する
- 対象 release が存在しない場合でも make 全体は失敗させない
- `ingress-nginx` namespace や deployment には触れない

### `cluster-restore`

`cluster-restore` を [Makefile](/home/shun/dev/keda-kind/Makefile) に追加する。

動作:

- `install-elasticmq`
- `install-postgresql`
- `install-keda`
- `install-enqueue`
- `install-dequeue`

を順に実行する。

要件:

- 既存 target を再利用し、install ロジックを重複させない
- 前提条件は user 管理とし、`build`、`kind-load`、`helm-deps` は含めない

## Documentation Changes

### README

[README.md](/home/shun/dev/keda-kind/README.md) に次を追記する。

- `cluster-clean` は `ingress-nginx` を残して app/DB/KEDA release だけ削除すること
- `cluster-restore` は事前に `make build`、`make kind-load`、`make helm-deps` を済ませた状態で使うこと

### TODO

[TODO.md](/home/shun/dev/keda-kind/TODO.md) に運用メモとして次を追記する。

- `make cluster-clean`
- `make cluster-restore`

必要なら確認コマンドも併記する。

## Verification

受け入れ確認は次の通り:

1. `make cluster-clean`
2. `env KUBECONFIG=$(pwd)/.cache/kubeconfig helm list -A`
3. `kubectl get pods -n ingress-nginx`
4. `make cluster-restore`
5. `env KUBECONFIG=$(pwd)/.cache/kubeconfig helm list -A`

期待結果:

- `cluster-clean` 後、対象 5 release が Helm list から消える
- `ingress-nginx` controller pod は残る
- `cluster-restore` 後、対象 5 release が戻る

## Risks And Mitigations

- `cluster-clean` で release 不在時に失敗するリスク
  - `helm uninstall ... || true` 相当で吸収する
- `cluster-restore` 実行時に image 未 build / 未 load で app が起動しないリスク
  - README/TODO に事前条件を明記する
- cleanup 対象が今後増えたときに漏れるリスク
  - README で「Helm 管理対象の app stack を消す target」であることを明示する
