CLUSTER_NAME ?= keda-kind
KUBECONFIG_PATH ?= $(CURDIR)/.cache/kubeconfig
PLATFORM ?= linux/amd64
IMAGE_TAG ?= dev
ENQUEUE_IMAGE ?= local/enqueue:$(IMAGE_TAG)
DEQUEUE_IMAGE ?= local/dequeue:$(IMAGE_TAG)
GOENV = GOPATH=$(CURDIR)/.cache/go GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod
DOCKER_ENV = DOCKER_CONFIG=$(CURDIR)/.cache/docker
HELM_ENV = HELM_CONFIG_HOME=$(CURDIR)/.cache/helm/config HELM_CACHE_HOME=$(CURDIR)/.cache/helm/cache HELM_DATA_HOME=$(CURDIR)/.cache/helm/data
KUBE_ENV = KUBECONFIG=$(KUBECONFIG_PATH)

.PHONY: test build-enqueue build-dequeue build kind-create kind-load ingress helm-deps helm-deps-argocd install-elasticmq install-postgresql install-keda install-keda-prod install-argocd install-argocd-apps install-enqueue install-enqueue-http install-dequeue keda-ready argocd-ready cluster-clean cluster-restore enqueue-scale-zero enqueue-scale-one compose-up compose-run-dequeue

test:
	env $(GOENV) go test ./...

build-enqueue:
	env $(DOCKER_ENV) docker buildx build --platform=$(PLATFORM) -f sample-app/docker/enqueue.Dockerfile -t $(ENQUEUE_IMAGE) --load .

build-dequeue:
	env $(DOCKER_ENV) docker buildx build --platform=$(PLATFORM) -f sample-app/docker/dequeue.Dockerfile -t $(DEQUEUE_IMAGE) --load .

build: build-enqueue build-dequeue

kind-create:
	env $(KUBE_ENV) kind create cluster --name $(CLUSTER_NAME) --config kind-config.yaml

kind-load:
	env $(KUBE_ENV) kind load docker-image --name $(CLUSTER_NAME) $(ENQUEUE_IMAGE)
	env $(KUBE_ENV) kind load docker-image --name $(CLUSTER_NAME) $(DEQUEUE_IMAGE)

ingress:
	env $(KUBE_ENV) kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
	env $(KUBE_ENV) kubectl patch deployment -n ingress-nginx ingress-nginx-controller --type merge -p '{"spec":{"template":{"spec":{"nodeSelector":{"kubernetes.io/os":"linux","ingress-ready":"true"}}}}}'
	env $(KUBE_ENV) kubectl rollout status deployment/ingress-nginx-controller -n ingress-nginx --timeout=180s

helm-deps:
	env $(KUBE_ENV) $(HELM_ENV) helm repo add kedacore https://kedacore.github.io/charts
	env $(KUBE_ENV) $(HELM_ENV) helm repo update
	env $(KUBE_ENV) $(HELM_ENV) helm dependency build manifest/keda-operator
	env $(KUBE_ENV) $(HELM_ENV) helm dependency build manifest/infra-bundle
	env $(KUBE_ENV) $(HELM_ENV) helm dependency build manifest/app-bundle

helm-deps-argocd:
	env $(KUBE_ENV) $(HELM_ENV) helm repo add argo https://argoproj.github.io/argo-helm
	env $(KUBE_ENV) $(HELM_ENV) helm repo update
	env $(KUBE_ENV) $(HELM_ENV) helm dependency build manifest/argocd

install-elasticmq:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install elasticmq ./manifest/elasticmq

install-postgresql:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install postgresql ./manifest/postgresql

install-keda:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install keda ./manifest/keda-operator -f manifest/keda-operator/values/develop.yaml -n keda --create-namespace

install-keda-prod:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install keda ./manifest/keda-operator -f manifest/keda-operator/values/production.yaml -n keda --create-namespace

install-argocd:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install argocd ./manifest/argocd -f manifest/argocd/values/develop.yaml -n argocd --create-namespace

install-argocd-apps:
	env $(KUBE_ENV) kubectl apply -f argocd/applicationsets/env-bundle.yaml

install-enqueue:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install enqueue ./manifest/enqueue-app -f manifest/enqueue-app/values/develop.yaml --set image.repository=local/enqueue --set image.tag=$(IMAGE_TAG)

install-enqueue-http:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install enqueue ./manifest/enqueue-app -f manifest/enqueue-app/values/develop.yaml --set image.repository=local/enqueue --set image.tag=$(IMAGE_TAG) --set mode=http --set httpPort=8080

enqueue-scale-zero:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade enqueue ./manifest/enqueue-app -f manifest/enqueue-app/values/develop.yaml --set image.repository=local/enqueue --set image.tag=$(IMAGE_TAG) --set replicaCount=0

enqueue-scale-one:
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade enqueue ./manifest/enqueue-app -f manifest/enqueue-app/values/develop.yaml --set image.repository=local/enqueue --set image.tag=$(IMAGE_TAG) --set replicaCount=1

install-dequeue: keda-ready
	env $(KUBE_ENV) $(HELM_ENV) helm upgrade --install dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml --set image.repository=local/dequeue --set image.tag=$(IMAGE_TAG)

cluster-clean:
	env $(KUBE_ENV) $(HELM_ENV) helm uninstall --ignore-not-found dequeue
	env $(KUBE_ENV) $(HELM_ENV) helm uninstall --ignore-not-found enqueue
	env $(KUBE_ENV) $(HELM_ENV) helm uninstall --ignore-not-found postgresql
	env $(KUBE_ENV) $(HELM_ENV) helm uninstall --ignore-not-found elasticmq
	env $(KUBE_ENV) $(HELM_ENV) helm uninstall --ignore-not-found keda -n keda

keda-ready:
	env $(KUBE_ENV) kubectl rollout status deployment/keda-operator -n keda --timeout=180s
	env $(KUBE_ENV) kubectl wait --for=condition=Established crd/scaledjobs.keda.sh --timeout=180s
	env $(KUBE_ENV) kubectl wait --for=condition=Established crd/triggerauthentications.keda.sh --timeout=180s

argocd-ready:
	env $(KUBE_ENV) kubectl rollout status deployment/argocd-server -n argocd --timeout=180s
	env $(KUBE_ENV) kubectl rollout status deployment/argocd-repo-server -n argocd --timeout=180s
	env $(KUBE_ENV) kubectl rollout status deployment/argocd-redis -n argocd --timeout=180s
	env $(KUBE_ENV) kubectl rollout status deployment/argocd-applicationset-controller -n argocd --timeout=180s
	env $(KUBE_ENV) kubectl rollout status deployment/argocd-notifications-controller -n argocd --timeout=180s
	env $(KUBE_ENV) kubectl rollout status deployment/argocd-commit-server -n argocd --timeout=180s
	env $(KUBE_ENV) kubectl rollout status statefulset/argocd-application-controller -n argocd --timeout=180s

cluster-restore: install-elasticmq install-postgresql install-keda install-enqueue install-dequeue

compose-up:
	env $(DOCKER_ENV) docker compose up -d elasticmq postgresql enqueue

compose-run-dequeue:
	env $(DOCKER_ENV) docker compose run --rm dequeue
