# Argo CD ApplicationSet + Helm 構成設計書

## 1. 文書概要

### 1.1 目的
本設計書は、Argo CD において **ApplicationSet + Cluster generator + Helm** を用いて、環境ごとにまとまった Application を生成し、その配下に複数サービスの実リソースをぶら下げて管理する構成を定義する。

本設計の目標は以下である。

- `dev-application` / `prod-application` のような **環境単位の Application** を生成する
- 各 Application の配下に、`service-a` / `service-b` など複数サービスの **Deployment / Service / ConfigMap 等の実リソース** を表示する
- デプロイ先クラスタは **Argo CD に登録された cluster Secret のラベル**で選択する
- 各サービスの Helm chart と、Argo CD の配布制御定義を分離する
- 環境ごとの values 選択は **Application 側の責務**として明示する

### 1.2 スコープ
本書で対象とするのは以下である。

- リポジトリ構成
- Helm chart 構成
- 環境バンドル chart 構成
- ApplicationSet 定義
- Cluster generator によるクラスタ選択
- 運用上の意図と制約
- 実装時の注意事項

本書では以下は対象外とする。

- CI パイプラインの詳細実装
- Secret 配布方法の詳細
- Argo CD の認証基盤設計
- 監査ログやポリシーエンジンの詳細

---

## 2. 背景と課題

### 2.1 背景
Argo CD で複数サービスを GitOps 管理する場合、次の2種類の見せ方がある。

1. **サービス単位で Application を分割する**
   - `service-a-dev`
   - `service-b-dev`
   - `service-a-prod`
   - `service-b-prod`

2. **環境単位で Application を束ねる**
   - `dev-application`
   - `prod-application`

今回求めるのは後者である。

### 2.2 解決したい課題
今回解決したい課題は以下である。

- 環境単位でまとまりを持った Argo CD 表示にしたい
- ただし親子 Application の多段ではなく、環境 Application の配下に実リソースを見せたい
- 配布対象クラスタは固定値ではなく、クラスタラベルで動的に選びたい
- 環境差分は values で切り替えたい
- サービス chart と Argo CD 制御定義を分離したい

### 2.3 多段 Application 構成を採用しない理由
`Application -> child Application -> resources` という多段構成は実装可能だが、今回の UI 要件とは一致しない。

多段構成では、親 Application 直下で主に見えるのは **子 Application** であり、`service-a` や `service-b` の Deployment / Service 等の実リソースは子 Application 側に属する。したがって、以下の見え方には直接ならない。

```text
dev-application
- service-a の Deployment / Service / ...
- service-b の Deployment / Service / ...
```

この見え方を実現するには、`dev-application` 自体がそれらの実リソースを **直接管理**する必要がある。そのため本設計では、環境ごとに 1 つの Application を生成し、その source を環境バンドル Helm chart に向ける構成を採用する。

---

## 3. 設計方針

### 3.1 採用方針
以下の方針を採用する。

1. **ApplicationSet を採用する**
   - 環境ごとの Application を自動生成するため
   - 配布対象クラスタをラベルセレクタで切り替えるため

2. **Cluster generator を採用する**
   - Argo CD に登録されたクラスタから対象クラスタを動的に選ぶため

3. **Helm を採用する**
   - 環境差分を values で管理するため
   - 複数サービスを bundle chart で束ねるため

4. **環境ごとの Application を採用する**
   - Argo CD UI 上で環境単位のまとまりを明示するため

5. **サービス chart と Argo CD 定義を分離する**
   - アプリケーション実装責務と配布制御責務を分離するため

### 3.2 責務分離
本設計では責務を以下のように分ける。

#### サービス chart
責務:
- そのサービス単体の Kubernetes リソースを生成する

含むもの:
- `Chart.yaml`
- `templates/`
- `values.yaml`
- `values/<env>.yaml`

#### 環境バンドル chart
責務:
- 複数サービスを 1 つの環境 Application に束ねる

含むもの:
- `Chart.yaml`
- `values.yaml`
- `values/<env>.yaml`
- 各サービスへの dependency 定義

#### ApplicationSet
責務:
- どのクラスタに、どの環境 Application を生成するかを制御する
- どの values ファイルを使うかを Application に渡す

### 3.3 values 選択責務
本設計では **どの values ファイルを読むかは Application 側で定義する**。

理由:
- Helm chart は「何を生成するか」を持つべきであり、「どの環境にどう配るか」は持つべきではない
- values ファイル選択は deploy 制御であり、Argo CD Application の責務に属する
- どの環境がどの values を参照しているかを Application 定義から明示的に確認できる

---

## 4. 全体アーキテクチャ

### 4.1 概要図

```text
Argo CD
└─ ApplicationSet (env-bundle)
   ├─ dev-application-<cluster>
   │  ├─ service-a Deployment / Service / ...
   │  └─ service-b Deployment / Service / ...
   └─ prod-application-<cluster>
      ├─ service-a Deployment / Service / ...
      └─ service-b Deployment / Service / ...
```

### 4.2 生成の流れ

1. Argo CD に複数クラスタを cluster Secret として登録する
2. 各 cluster Secret に `environment=dev` や `environment=prod` などのラベルを付与する
3. ApplicationSet が Cluster generator でラベル一致したクラスタを列挙する
4. generator ごとに `env` と `valuesFile` を template に渡す
5. ApplicationSet が `dev-application-<cluster>` / `prod-application-<cluster>` を生成する
6. 各 Application は `manifests/env-bundle` chart を Helm でレンダリングする
7. `env-bundle` chart が `service-a` / `service-b` chart を dependency としてまとめて描画する
8. Argo CD UI では環境 Application の配下に各サービスの実リソースが見える

---

## 5. リポジトリ構成

```text
repo/
├─ manifests/
│  ├─ service-a/
│  │  ├─ Chart.yaml
│  │  ├─ values.yaml
│  │  ├─ values/
│  │  │  ├─ dev.yaml
│  │  │  └─ prod.yaml
│  │  └─ templates/
│  │     ├─ _helpers.tpl
│  │     ├─ deployment.yaml
│  │     └─ service.yaml
│  ├─ service-b/
│  │  ├─ Chart.yaml
│  │  ├─ values.yaml
│  │  ├─ values/
│  │  │  ├─ dev.yaml
│  │  │  └─ prod.yaml
│  │  └─ templates/
│  │     ├─ _helpers.tpl
│  │     ├─ deployment.yaml
│  │     └─ service.yaml
│  └─ env-bundle/
│     ├─ Chart.yaml
│     ├─ values.yaml
│     └─ values/
│        ├─ dev.yaml
│        └─ prod.yaml
└─ argocd/
   └─ applicationsets/
      └─ env-bundle.yaml
```

### 5.1 ディレクトリ意図

#### `manifests/service-a`, `manifests/service-b`
各サービス単体の Helm chart を格納する。  
ここには Kubernetes リソース定義と、そのサービス固有の values を置く。

#### `manifests/env-bundle`
複数サービスを 1 つの環境 Application に束ねるための親 chart を格納する。  
Argo CD から直接参照される chart はこれである。

#### `argocd/applicationsets`
Argo CD 制御定義を格納する。  
ApplicationSet などの GitOps 制御オブジェクトは、サービス chart とは責務が異なるため分離する。

---

## 6. 各コンポーネント詳細

# 6.1 service-a chart

## 6.1.1 `manifests/service-a/Chart.yaml`

```yaml
apiVersion: v2
name: service-a
description: Helm chart for service-a
type: application
version: 0.1.0
appVersion: "1.0.0"
```

### 意図
- `service-a` 単体の chart であることを明示する
- chart version と app version を分離する

---

## 6.1.2 `manifests/service-a/values.yaml`

```yaml
nameOverride: ""
fullnameOverride: ""

replicaCount: 1

image:
  repository: ghcr.io/example/service-a
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80

containerPort: 8080

resources: {}

env: []
```

### 意図
- デフォルト値を置く
- 環境差分は `values/dev.yaml`, `values/prod.yaml` で上書きする
- chart 単体でもレンダリング可能な最小値を持つ

---

## 6.1.3 `manifests/service-a/values/dev.yaml`

```yaml
replicaCount: 1

image:
  tag: dev-latest

service:
  port: 8080

env:
  - name: APP_ENV
    value: dev
```

### 意図
- dev 環境に必要な最小差分だけを持つ
- 共通値は `values.yaml` に残し、差分のみを上書きする

---

## 6.1.4 `manifests/service-a/values/prod.yaml`

```yaml
replicaCount: 3

image:
  tag: "1.0.0"

service:
  port: 8080

env:
  - name: APP_ENV
    value: prod
```

### 意図
- 本番ではレプリカ数や image tag を固定する
- prod 特有の設定のみを定義する

---

## 6.1.5 `manifests/service-a/templates/_helpers.tpl`

```yaml
{{- define "service-a.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "service-a.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "service-a.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "service-a.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" -}}
{{- end -}}
```

### 意図
- resource 名の生成ロジックを共通化する
- Release 名を prefix にして、bundle chart 配下でも一意性を保ちやすくする

---

## 6.1.6 `manifests/service-a/templates/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "service-a.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "service-a.name" . }}
    helm.sh/chart: {{ include "service-a.chart" . }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "service-a.name" . }}
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ include "service-a.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
        - name: service-a
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - containerPort: {{ .Values.containerPort }}
          env:
{{ toYaml .Values.env | indent 12 }}
          resources:
{{ toYaml .Values.resources | indent 12 }}
```

### 意図
- 標準的な Deployment を定義する
- values から replica 数や image tag を上書きできるようにする
- env 変数や resources を values 化して環境別調整を可能にする

---

## 6.1.7 `manifests/service-a/templates/service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "service-a.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "service-a.name" . }}
    helm.sh/chart: {{ include "service-a.chart" . }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
spec:
  type: {{ .Values.service.type }}
  selector:
    app.kubernetes.io/name: {{ include "service-a.name" . }}
    app.kubernetes.io/instance: {{ .Release.Name }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.containerPort }}
      protocol: TCP
      name: http
```

### 意図
- Deployment と対応する Service を生成する
- port や type を values 化して環境調整可能にする

---

# 6.2 service-b chart

## 6.2.1 `manifests/service-b/Chart.yaml`

```yaml
apiVersion: v2
name: service-b
description: Helm chart for service-b
type: application
version: 0.1.0
appVersion: "1.0.0"
```

## 6.2.2 `manifests/service-b/values.yaml`

```yaml
nameOverride: ""
fullnameOverride: ""

replicaCount: 1

image:
  repository: ghcr.io/example/service-b
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80

containerPort: 8081

resources: {}

env: []
```

## 6.2.3 `manifests/service-b/values/dev.yaml`

```yaml
replicaCount: 1

image:
  tag: dev-latest

service:
  port: 8081

env:
  - name: APP_ENV
    value: dev
```

## 6.2.4 `manifests/service-b/values/prod.yaml`

```yaml
replicaCount: 2

image:
  tag: "2.0.0"

service:
  port: 8081

env:
  - name: APP_ENV
    value: prod
```

## 6.2.5 `manifests/service-b/templates/_helpers.tpl`

```yaml
{{- define "service-b.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "service-b.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "service-b.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "service-b.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" -}}
{{- end -}}
```

## 6.2.6 `manifests/service-b/templates/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "service-b.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "service-b.name" . }}
    helm.sh/chart: {{ include "service-b.chart" . }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "service-b.name" . }}
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ include "service-b.name" . }}
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
        - name: service-b
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - containerPort: {{ .Values.containerPort }}
          env:
{{ toYaml .Values.env | indent 12 }}
          resources:
{{ toYaml .Values.resources | indent 12 }}
```

## 6.2.7 `manifests/service-b/templates/service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "service-b.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "service-b.name" . }}
    helm.sh/chart: {{ include "service-b.chart" . }}
    app.kubernetes.io/instance: {{ .Release.Name }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
spec:
  type: {{ .Values.service.type }}
  selector:
    app.kubernetes.io/name: {{ include "service-b.name" . }}
    app.kubernetes.io/instance: {{ .Release.Name }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.containerPort }}
      protocol: TCP
      name: http
```

### 意図
service-a と同様の設計とし、サービス間で chart 構造を揃える。  
これにより、レビュー・運用・自動化のパターンを統一できる。

---

# 6.3 env-bundle chart

## 6.3.1 `manifests/env-bundle/Chart.yaml`

```yaml
apiVersion: v2
name: env-bundle
description: Bundle chart for one environment
type: application
version: 0.1.0
appVersion: "1.0.0"

dependencies:
  - name: service-a
    version: 0.1.0
    repository: "file://../service-a"
  - name: service-b
    version: 0.1.0
    repository: "file://../service-b"
```

### 意図
- `dev-application` / `prod-application` が直接参照する親 chart とする
- `service-a` / `service-b` を dependency として束ねる
- Argo CD 側からはこの chart を 1 つ指定するだけで複数サービスをまとめて管理できるようにする

### 採用理由
この bundle chart を採用することで、Argo CD 上の 1 Application の配下に、複数サービスの実リソースを直接見せることができる。  
もし各サービスを個別 Application にすると、親の下に見えるのは子 Application であり、実リソースの直接表示にはならない。

---

## 6.3.2 `manifests/env-bundle/values.yaml`

```yaml
service-a: {}
service-b: {}
```

### 意図
- 依存チャートに値を受け渡すためのルートを明示する
- デフォルトでは上書きなしとし、環境差分は `values/<env>.yaml` に寄せる

---

## 6.3.3 `manifests/env-bundle/values/dev.yaml`

```yaml
service-a:
  replicaCount: 1
  image:
    tag: dev-latest
  env:
    - name: APP_ENV
      value: dev

service-b:
  replicaCount: 1
  image:
    tag: dev-latest
  env:
    - name: APP_ENV
      value: dev
```

### 意図
- dev 環境で service-a / service-b 双方に適用する値を 1 ファイルに集約する
- 環境単位の Application に対して必要な差分を 1 か所で把握できるようにする

---

## 6.3.4 `manifests/env-bundle/values/prod.yaml`

```yaml
service-a:
  replicaCount: 3
  image:
    tag: "1.0.0"
  env:
    - name: APP_ENV
      value: prod

service-b:
  replicaCount: 2
  image:
    tag: "2.0.0"
  env:
    - name: APP_ENV
      value: prod
```

### 意図
- prod 用の値を束ねる
- 本番では明示的な image tag を指定し、意図しない更新を防ぐ
- 環境 Application ごとに参照する values を切り替える構成にする

---

## 6.3.5 なぜ bundle chart の values に環境差分を置くのか
サービス chart 自体にも `values/dev.yaml`, `values/prod.yaml` は持てるが、今回の要件では環境単位 Application の source は `env-bundle` である。したがって、Application が直接選択する values は `env-bundle` 側に持たせる方が自然である。

この設計により、

- Application:
  - どの環境 values を読むかを決める
- env-bundle:
  - その環境で各サービスにどう値を渡すかを持つ
- service chart:
  - 最終的な manifest 生成ロジックを持つ

という階層分離が成立する。

---

# 6.4 ApplicationSet

## 6.4.1 `argocd/applicationsets/env-bundle.yaml`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: env-bundle
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions:
    - missingkey=error

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

  template:
    metadata:
      name: '{{.values.env}}-application-{{.nameNormalized}}'
      namespace: argocd
      labels:
        env: '{{.values.env}}'
        cluster: '{{.nameNormalized}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/example/platform-gitops.git
        targetRevision: main
        path: manifests/env-bundle
        helm:
          valueFiles:
            - '{{.values.valuesFile}}'
      destination:
        server: '{{.server}}'
        namespace: platform
      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

### 意図
- `environment=dev` ラベルのクラスタに対して dev 用 Application を生成する
- `environment=prod` ラベルのクラスタに対して prod 用 Application を生成する
- values ファイル選択は generator から template に渡す
- Application 名にはクラスタ名を含め、同一 env に複数クラスタがある場合の衝突を回避する

### なぜ `valueFiles` を Application 側に置くのか
これは本設計の重要な設計意図である。

- chart 側は values ファイルを提供する
- どれを採用するかは deploy 制御の問題である
- したがって、`helm.valueFiles` は Application 側に置く

この構成により、Argo CD 定義を見れば「その Application がどの環境値でレンダリングされるか」が明確になる。

### `goTemplate: true` を使う理由
- generator から渡された `env`, `valuesFile`, `server` を柔軟に扱うため
- `missingkey=error` によって typo や値不足を早期に検知するため

### `CreateNamespace=true` の意図
- destination namespace が未作成でも同期可能にするため
- 初回 bootstrap 時の手作業を減らすため

---

## 6.4.2 Application 名の考え方
現在の例では Application 名は以下になる。

- `dev-application-dev-cluster`
- `prod-application-prod-cluster`

これは **Cluster generator がクラスタごとに Application を 1 個生成する** 性質に合わせた設計である。  
同一 env に複数クラスタが存在する場合、単純な `dev-application` という名前では衝突する。

### 例外
もし本当に以下が保証されるなら、

- `environment=dev` のクラスタは常に 1 個
- `environment=prod` のクラスタは常に 1 個

Application 名は次でもよい。

```yaml
name: '{{.values.env}}-application'
```

ただし将来クラスタが増える可能性があるなら、初期設計からクラスタ名込みにしておく方が安全である。

---

## 6.4.3 destination namespace の考え方
例では固定で `platform` namespace を使っている。

```yaml
destination:
  namespace: platform
```

これは「環境単位 Application の下に複数サービスをまとめ、同一 namespace に配置する」前提である。

別案としては以下もありえる。

- `platform-dev`, `platform-prod`
- サービスごとに namespace 分離
- env ごと + team ごとに namespace 分離

ただし今回の要件の本質は「環境 Application に実リソースを束ねて見せること」であるため、namespace 戦略は別途拡張可能な可変項目とする。

---

## 6.5 cluster Secret ラベル設計

### 6.5.1 dev クラスタ Secret 例

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dev-cluster-secret
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: cluster
    environment: dev
type: Opaque
stringData:
  name: dev-cluster
  server: https://dev.example.local
  config: |
    {
      "bearerToken": "REDACTED",
      "tlsClientConfig": {
        "insecure": false
      }
    }
```

### 6.5.2 prod クラスタ Secret 例

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: prod-cluster-secret
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: cluster
    environment: prod
type: Opaque
stringData:
  name: prod-cluster
  server: https://prod.example.local
  config: |
    {
      "bearerToken": "REDACTED",
      "tlsClientConfig": {
        "insecure": false
      }
    }
```

### 意図
- Cluster generator は cluster Secret のラベルを見て対象を選ぶ
- `environment` ラベルは ApplicationSet の selector 条件と一致する
- 将来的に以下のようなラベル拡張が可能である
  - `region: ap-northeast-1`
  - `tier: shared`
  - `team: platform`

### 推奨
`environment` のほかに、将来的な拡張を考えるなら以下のようなラベルセットを推奨する。

- `environment`
- `region`
- `tier`
- `owner`

---

## 7. 実装手順

### 7.1 リポジトリ作成
1. `manifests/service-a` を作成する
2. `manifests/service-b` を作成する
3. `manifests/env-bundle` を作成する
4. `argocd/applicationsets` を作成する

### 7.2 service chart 実装
1. `Chart.yaml` を作成する
2. `values.yaml` を作成する
3. `templates/` を作成する
4. `values/dev.yaml`, `values/prod.yaml` を作成する
5. `helm template` で単体レンダリング検証する

### 7.3 env-bundle chart 実装
1. `dependencies` に各サービス chart を定義する
2. `values/dev.yaml`, `values/prod.yaml` を作成する
3. `helm dependency update manifests/env-bundle` を実行する
4. `helm template` で bundle 単位レンダリング検証する

### 7.4 Argo CD 側
1. 対象クラスタを Argo CD に登録する
2. cluster Secret に `environment` ラベルを付与する
3. `ApplicationSet` を apply する
4. 生成された Application を確認する
5. 各 Application の配下に service-a / service-b のリソースが見えることを確認する

---

## 8. 実装時の確認項目

### 8.1 Helm dependency 検証
`env-bundle` は local dependency を使っているため、以下を CI で検証することを推奨する。

```bash
helm dependency update manifests/env-bundle
helm template test manifests/env-bundle -f manifests/env-bundle/values/dev.yaml
helm template test manifests/env-bundle -f manifests/env-bundle/values/prod.yaml
```

### 8.2 values の責務確認
以下を守ること。

- サービス固有の default 値は各 service chart の `values.yaml`
- 環境差分は env-bundle の `values/<env>.yaml`
- どの環境値を採用するかは Application 側

### 8.3 命名の一貫性
以下を統一すること。

- service 名
- env 名
- cluster label 名
- Application 名の命名規則

---

## 9. 運用設計

### 9.1 運用時の見え方
Argo CD 上では次のようになる。

```text
dev-application-dev-cluster
- Deployment / dev-application-dev-cluster-service-a
- Service / dev-application-dev-cluster-service-a
- Deployment / dev-application-dev-cluster-service-b
- Service / dev-application-dev-cluster-service-b

prod-application-prod-cluster
- Deployment / prod-application-prod-cluster-service-a
- Service / prod-application-prod-cluster-service-a
- Deployment / prod-application-prod-cluster-service-b
- Service / prod-application-prod-cluster-service-b
```

### 9.2 変更パターン
#### service-a のテンプレート変更
- `manifests/service-a/templates/*` を変更
- dev/prod 両環境の service-a に反映される

#### dev だけ replica 数変更
- `manifests/env-bundle/values/dev.yaml` を変更
- dev Application にのみ影響する

#### prod 配布クラスタ追加
- 新しい cluster Secret を登録
- `environment=prod` を付与
- ApplicationSet が自動的に新しい prod Application を生成する

---

## 10. 制約と注意点

### 10.1 同一 env に複数クラスタがある場合
Cluster generator はクラスタごとに Application を生成する。  
そのため、厳密に `dev-application` 1個だけに固定したい場合は、同一 env ラベルのクラスタを 1 個に制限する必要がある。

### 10.2 1 Application に複数サービスを束ねるトレードオフ
利点:
- 環境単位で見やすい
- 環境単位の変更をまとめて反映しやすい

欠点:
- サービス単位の同期やロールバック粒度は粗くなる
- ある環境内で service-a だけ別サイクルで反映したい場合に向かない

### 10.3 service chart の values/ 配置について
今回の最終レンダリングで直接使う values は env-bundle 側である。  
service chart 側の `values/dev.yaml`, `values/prod.yaml` は以下用途として残す。

- サービス単体でのローカル検証
- 将来 service 単位 Application に戻す場合の再利用
- chart 利用者に対する参考値

ただし、現行運用で Argo CD が直接読むのは env-bundle 側の values とする。

### 10.4 namespace 戦略
本設計では namespace を簡略化のため固定している。  
本番運用では以下のどれかを別途定義する必要がある。

- 環境ごと固定 namespace
- サービスごと namespace 分離
- team / environment の複合 namespace

---

## 11. 拡張案

### 11.1 AppProject 導入
環境ごと / チームごとに deploy 先や repo を制限したい場合は、`AppProject` を導入する。

### 11.2 RollingSync 導入
大量クラスタへ段階的反映したい場合は、ApplicationSet の progressive sync を利用する。

### 11.3 values の外部化
将来的に chart と設定を別 repo に分離したい場合は、Argo CD の multiple sources で chart と values を分離することを検討する。

---

## 12. 採用判断まとめ

本設計は次の要件に適している。

- 環境単位の Application にしたい
- 配布対象クラスタをラベル選択にしたい
- Helm を使いたい
- 環境ごとの values 選択を Application 側に持たせたい
- サービス実体と Argo CD 制御定義を分離したい

本設計を採用することで、以下が実現できる。

- `dev-application` / `prod-application` 系の環境単位表示
- 各 Application の配下に複数サービスの実リソース表示
- cluster Secret ラベルによる動的配布
- values 選択責務の明確化
- service chart / bundle chart / ApplicationSet の責務分離

---

## 13. 実装用ファイル一覧

### 必須ファイル
- `manifests/service-a/Chart.yaml`
- `manifests/service-a/values.yaml`
- `manifests/service-a/values/dev.yaml`
- `manifests/service-a/values/prod.yaml`
- `manifests/service-a/templates/_helpers.tpl`
- `manifests/service-a/templates/deployment.yaml`
- `manifests/service-a/templates/service.yaml`

- `manifests/service-b/Chart.yaml`
- `manifests/service-b/values.yaml`
- `manifests/service-b/values/dev.yaml`
- `manifests/service-b/values/prod.yaml`
- `manifests/service-b/templates/_helpers.tpl`
- `manifests/service-b/templates/deployment.yaml`
- `manifests/service-b/templates/service.yaml`

- `manifests/env-bundle/Chart.yaml`
- `manifests/env-bundle/values.yaml`
- `manifests/env-bundle/values/dev.yaml`
- `manifests/env-bundle/values/prod.yaml`

- `argocd/applicationsets/env-bundle.yaml`

---

## 14. 参考コマンド

### ローカル検証
```bash
helm template local manifests/service-a -f manifests/service-a/values/dev.yaml
helm template local manifests/service-b -f manifests/service-b/values/dev.yaml

helm dependency update manifests/env-bundle
helm template dev manifests/env-bundle -f manifests/env-bundle/values/dev.yaml
helm template prod manifests/env-bundle -f manifests/env-bundle/values/prod.yaml
```

### 適用
```bash
kubectl apply -f argocd/applicationsets/env-bundle.yaml
```

---

## 15. 最終結論

本設計の中核は次の1文に要約できる。

> 環境ごとに 1 つの Application を ApplicationSet で生成し、その Application が Helm の env-bundle chart を直接参照することで、配下に複数サービスの実リソースをぶら下げて見せる。

この方針により、UI の見え方、責務分離、環境差分管理、クラスタ動的選択を同時に満たすことができる。
