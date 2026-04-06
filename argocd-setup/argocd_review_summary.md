# Argo CD ApplicationSet + Helm 構成レビュー用サマリ

## 結論
本構成では、**ApplicationSet + Cluster generator + Helm bundle chart** を用いて、環境単位の Application を生成する。

Argo CD 上の見え方は次のとおり。

```text
dev-application-<cluster>
- service-a の Deployment / Service / ...
- service-b の Deployment / Service / ...

prod-application-<cluster>
- service-a の Deployment / Service / ...
- service-b の Deployment / Service / ...
```

## この構成を採用する理由
- 環境単位でまとまった Application を表示できる
- 親子 Application の多段構成にせず、環境 Application の配下に実リソースを見せられる
- 配布先クラスタを cluster Secret のラベルで選択できる
- 環境差分を Helm values で管理できる
- サービス chart と Argo CD 制御定義を分離できる

## 基本方針
- 各サービスは `manifests/<service>/` に Helm chart として配置する
- `manifests/env-bundle/` に親 Helm chart を置き、`service-a` と `service-b` を dependency として束ねる
- `argocd/applicationsets/` に ApplicationSet を置く
- どの values を使うかは `Application` 側の `helm.valueFiles` で指定する

## ディレクトリ構成
```text
repo/
├─ manifests/
│  ├─ service-a/
│  ├─ service-b/
│  └─ env-bundle/
└─ argocd/
   └─ applicationsets/
      └─ env-bundle.yaml
```

## 責務分離
### `manifests/service-a`, `manifests/service-b`
- サービス単体の Kubernetes リソースを生成する責務

### `manifests/env-bundle`
- 複数サービスを 1 つの環境 Application に束ねる責務

### `argocd/applicationsets`
- どのクラスタに、どの values で Application を生成するかを制御する責務

## values 選択を Application 側に置く理由
- Helm chart は「何を生成するか」を持つべき
- Application は「どの環境に、どの値で配るか」を持つべき
- Argo CD 定義を見れば、どの values が使われるかが分かる

## ApplicationSet の考え方
Cluster generator で cluster Secret のラベルを見てクラスタを選ぶ。

例:
- `environment=dev` → dev 用 Application を生成
- `environment=prod` → prod 用 Application を生成

代表例:

```yaml
generators:
  - clusters:
      selector:
        matchLabels:
          environment: dev
      values:
        env: dev
        valuesFile: values/dev.yaml
  - clusters:
      selector:
        matchLabels:
          environment: prod
      values:
        env: prod
        valuesFile: values/prod.yaml
```

## 想定される Application
- `dev-application-dev-cluster`
- `prod-application-prod-cluster`

同じ env に複数クラスタがある場合、Cluster generator はクラスタごとに Application を生成するため、Application 名にはクラスタ名を含める。

## 利点
- UI 上で環境単位にまとまる
- 環境差分が `env-bundle/values/<env>.yaml` に集約される
- 対象クラスタ追加時は cluster Secret を追加してラベルを付与するだけでよい
- service chart と配布制御を独立変更できる

## トレードオフ
- 1 Application に複数サービスを束ねるため、同期・ロールバック粒度はサービス単位より粗い
- 同一 env に複数クラスタがある場合、Application は env ごとに 1 個ではなくクラスタごとに増える
- namespace 戦略は別途設計が必要

## 採用判断
以下を重視するなら本構成は適切である。

- 環境単位の見え方
- 実リソースを Application 配下に直接表示
- クラスタラベルによる動的配布
- Helm による環境差分管理
- サービス実装と配布制御の責務分離
