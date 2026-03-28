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

.PHONY: test build-enqueue build-dequeue build kind-create kind-load ingress helm-deps install-elasticmq install-postgresql install-keda install-keda-prod install-enqueue install-dequeue compose-up compose-run-dequeue

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
	env $(HELM_ENV) helm repo add kedacore https://kedacore.github.io/charts
	env $(HELM_ENV) helm repo update
	env $(HELM_ENV) helm dependency build manifest/keda-operator

install-elasticmq:
	env $(HELM_ENV) helm upgrade --install elasticmq ./manifest/elasticmq

install-postgresql:
	env $(HELM_ENV) helm upgrade --install postgresql ./manifest/postgresql

install-keda:
	env $(HELM_ENV) helm upgrade --install keda ./manifest/keda-operator -f manifest/keda-operator/values/develop.yaml -n keda --create-namespace

install-keda-prod:
	env $(HELM_ENV) helm upgrade --install keda ./manifest/keda-operator -f manifest/keda-operator/values/production.yaml -n keda --create-namespace

install-enqueue:
	env $(HELM_ENV) helm upgrade --install enqueue ./manifest/enqueue-app -f manifest/enqueue-app/values/develop.yaml --set image.repository=local/enqueue --set image.tag=$(IMAGE_TAG)

install-dequeue:
	env $(HELM_ENV) helm upgrade --install dequeue ./manifest/dequeue-app -f manifest/dequeue-app/values/develop.yaml --set image.repository=local/dequeue --set image.tag=$(IMAGE_TAG)

compose-up:
	env $(DOCKER_ENV) docker compose up -d elasticmq postgresql enqueue

compose-run-dequeue:
	env $(DOCKER_ENV) docker compose run --rm dequeue
