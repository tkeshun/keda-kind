# ArgoCD ApplicationSet Helm Source 移行設計

## 目的

既存の静的 `Application` 3件を、`ApplicationSet` が生成する `Application` に移行する。生成される `Application` は、これまでと同じく既存の Helm chart を `source.path` で参照する。

## 方針

- `ApplicationSet` 自体は `argocd/applicationsets/env-bundle.yaml` に素の YAML として置く。
- generator は初期段階では `list` を使い、kind の in-cluster 向け定義を明示する。
- 生成対象は `keda-operator`、`infra-core`、`sample-app` の3件にする。
- 各 `Application` の `source.path` は既存chartの `manifest/keda-operator`、`manifest/infra-bundle`、`manifest/app-bundle` を維持する。
- 旧 `argocd/applications/*.yaml` は削除し、repo上の正を `ApplicationSet` に寄せる。

## 生成対象

| name | chart path | values file | destination namespace |
| --- | --- | --- | --- |
| `keda-operator` | `manifest/keda-operator` | `values/develop.yaml` | `keda` |
| `infra-core` | `manifest/infra-bundle` | `values/develop.yaml` | `default` |
| `sample-app` | `manifest/app-bundle` | `values/develop.yaml` | `default` |

## 制限

- `repoURL` は既存と同じく置換前提のプレースホルダーにする。
- `targetRevision` は既存と同じく初期値 `HEAD` にする。GitHub同期時に `main` へ切り替える。
- ApplicationSet CRDが未導入の環境では、`kubectl apply --dry-run` はresource mapping解決に失敗する。CRD導入前の構文確認はYAML parserで行い、実クラスタ上のdry-runはArgoCD導入後に行う。

## 検証

- `python3` の YAML parser で `argocd/applicationsets/env-bundle.yaml` の構文を確認する。
- `helm template` で3つの参照先chartがrenderできることを確認する。
- `sample-app/layout` testでApplicationSetパスが保護されることを確認する。
- `make test` でGo全体の回帰がないことを確認する。
