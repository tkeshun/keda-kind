# リポジトリ配置用雛形一式

この文書は、そのままリポジトリへ配置できる最小雛形を示す。

## 1. ディレクトリ構成

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

---

## 2. 配置ファイル

### `manifests/service-a/Chart.yaml`
```yaml
apiVersion: v2
name: service-a
description: Helm chart for service-a
type: application
version: 0.1.0
appVersion: "1.0.0"
```

### `manifests/service-a/values.yaml`
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

### `manifests/service-a/values/dev.yaml`
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

### `manifests/service-a/values/prod.yaml`
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

### `manifests/service-a/templates/_helpers.tpl`
```tpl
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

### `manifests/service-a/templates/deployment.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "service-a.fullname" . }}
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

### `manifests/service-a/templates/service.yaml`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "service-a.fullname" . }}
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

---

### `manifests/service-b/Chart.yaml`
```yaml
apiVersion: v2
name: service-b
description: Helm chart for service-b
type: application
version: 0.1.0
appVersion: "1.0.0"
```

### `manifests/service-b/values.yaml`
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

### `manifests/service-b/values/dev.yaml`
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

### `manifests/service-b/values/prod.yaml`
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

### `manifests/service-b/templates/_helpers.tpl`
```tpl
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

### `manifests/service-b/templates/deployment.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "service-b.fullname" . }}
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

### `manifests/service-b/templates/service.yaml`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "service-b.fullname" . }}
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

---

### `manifests/env-bundle/Chart.yaml`
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

### `manifests/env-bundle/values.yaml`
```yaml
service-a: {}
service-b: {}
```

### `manifests/env-bundle/values/dev.yaml`
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

### `manifests/env-bundle/values/prod.yaml`
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

---

### `argocd/applicationsets/env-bundle.yaml`
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

---

## 3. クラスタラベル例

### dev
```yaml
metadata:
  labels:
    argocd.argoproj.io/secret-type: cluster
    environment: dev
```

### prod
```yaml
metadata:
  labels:
    argocd.argoproj.io/secret-type: cluster
    environment: prod
```

---

## 4. 使い方

### 依存関係更新
```bash
helm dependency update manifests/env-bundle
```

### ローカルレンダリング確認
```bash
helm template dev manifests/env-bundle -f manifests/env-bundle/values/dev.yaml
helm template prod manifests/env-bundle -f manifests/env-bundle/values/prod.yaml
```

### ApplicationSet 適用
```bash
kubectl apply -f argocd/applicationsets/env-bundle.yaml
```

---

## 5. 最初に変える場所

最初に自分の環境向けに変更する箇所は以下。

- `ghcr.io/example/service-a`
- `ghcr.io/example/service-b`
- `https://github.com/example/platform-gitops.git`
- `destination.namespace`
- `environment` ラベル値
- prod 用 image tag
