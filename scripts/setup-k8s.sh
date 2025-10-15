#!/bin/bash
set -euo pipefail

echo "[Step 1] Installing required CLI tools (kubectl)"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

KUBECTL_URL="https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/${OS}/${ARCH}/kubectl"
if ! command -v kubectl &> /dev/null; then
  echo "Installing kubectl..."
  curl -LO "$KUBECTL_URL"
  chmod +x kubectl
  sudo mv kubectl /usr/local/bin/
else
  echo "kubectl already installed"
fi

echo "[Step 2] Installing Knative Serving v1.17"
KNATIVE_VERSION="knative-v1.17.0"

# Best-effort pre-pull (можно пропустить, если сеть недоступна)
echo "[Step 2a] Pulling Knative images (best effort)"
docker pull gcr.io/knative-releases/knative.dev/serving/cmd/activator@sha256:cd4bb3af998f4199ea760718a309f50d1bcc9d5c4a1c5446684a6a0115a7aad5 || true
docker pull gcr.io/knative-releases/knative.dev/serving/cmd/autoscaler@sha256:ac1a83ba7c278ce9482b7bbfffe00e266f657b7d2356daed88ffe666bc68978e || true
docker pull gcr.io/knative-releases/knative.dev/serving/cmd/controller@sha256:df24c6d3e20bc22a691fcd8db6df25a66c67498abd38a8a56e8847cb6bfb875b || true
docker pull gcr.io/knative-releases/knative.dev/serving/cmd/webhook@sha256:d842f05a1b05b1805021b9c0657783b4721e79dc96c5b58dc206998c7062d9d9 || true

kubectl apply -f https://github.com/knative/serving/releases/download/${KNATIVE_VERSION}/serving-crds.yaml
kubectl apply -f https://github.com/knative/serving/releases/download/${KNATIVE_VERSION}/serving-core.yaml

echo "[Step 3] Installing/Updating Kourier ingress"
kubectl apply -f https://github.com/knative/net-kourier/releases/download/${KNATIVE_VERSION}/kourier.yaml

echo "[Step 4] Waiting for Knative Serving and Kourier to be ready"
kubectl wait deployment --all --timeout=300s --for=condition=Available -n knative-serving
kubectl wait deployment --all --timeout=300s --for=condition=Available -n kourier-system

echo "[Step 5] Configure domain knative.demo.com"
kubectl patch configmap/config-network -n knative-serving --type merge \
  -p '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
kubectl patch configmap/config-domain -n knative-serving --type merge \
  -p '{"data":{"knative.demo.com":""}}'

echo "[Step 6] (Optional) Deploy sample echo service"
kubectl delete ksvc echo -n default --ignore-not-found
cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: echo
  namespace: default
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "5"
        autoscaling.knative.dev/target: "50"
        autoscaling.knative.dev/class: "kpa.autoscaling.knative.dev"
        autoscaling.knative.dev/metric: "rps"
        networking.knative.dev/ingress.class: "kourier.ingress.networking.knative.dev"
    spec:
      containers:
        - image: ealen/echo-server:latest
          ports:
            - containerPort: 80
EOF
kubectl wait ksvc echo --timeout=300s --for=condition=Ready || true
echo "[Check] Echo test:"
curl -s -H "Host: echo.default.knative.demo.com" "http://localhost/" || true

echo "[Step 7] Start local Docker registry"
docker run -d -p 5000:5000 --name registry registry:2 || true

echo "[Step 8] Deploy infrastructure services (Redis, Postgres)"
kubectl apply -f deployment/redis/ || true
kubectl apply -f deployment/postgres/ || true

echo "[Step 9] Deploy waiter service"
kubectl delete ksvc waiter -n default --ignore-not-found
docker build -t dev.local/waiter:latest ./waiter-service
kubectl apply -f ./waiter-service/service.yml
kubectl wait ksvc waiter --timeout=300s --for=condition=Ready
# kubectl apply -f waiter-service/k8s/service.yml
# kubectl wait ksvc waiter --timeout=300s --for=condition=Ready

echo "[Step 10] Smoke test waiter"
curl -s -H "Host: waiter.default.knative.demo.com" "http://localhost/invoke?sleep_ms=200&mem_mb=50" || true

echo "[Done] Environment is ready."
