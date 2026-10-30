IMAGE_NAME := http-server
IMAGE_TAG := v1.0
IMAGE := $(IMAGE_NAME):$(IMAGE_TAG)
KIND_CLUSTER_NAME := http-from-scratch-training
KIND_CONFIG := kind-config.yaml
GATEWAY_VERSION := v1.6.3

deploy-local: image
	@if ! kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
        kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG); \
    fi
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER_NAME)
	@if ! kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1; then \
        kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_VERSION)/experimental-install.yaml; \
    fi
	@if ! kubectl get namespace envoy-gateway-system >/dev/null 2>&1; then \
        kubectl apply --server-side -f https://github.com/envoyproxy/gateway/releases/latest/download/install.yaml; \
    fi
	kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available
	kubectl apply -f deployments/
	kubectl wait --timeout=2m -n default gateway/main-gateway --for=jsonpath='{.status.conditions[?(@.type=="Programmed")].status}'=True
image:
	docker build -t $(IMAGE) .

delete-deployments:
	kubectl delete -f deployments/

clean:
	kind delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: image deploy-local delete-deployments clean