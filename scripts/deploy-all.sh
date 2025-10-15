#!/bin/bash
set -euo pipefail

echo "=== FaaS Billing System Deployment ==="

# 1. Подготовка инфраструктуры
echo "Step 1: Setting up K8s and Knative"
./scripts/setup-k8s.sh

echo "Step 2: Setting up monitoring"
./scripts/setup-prometheus.sh

# 2. Сборка образов
echo "Step 3: Building container images"
docker build -t waiter:latest ./waiter-service/
docker tag waiter:latest localhost:5000/waiter:latest
docker push localhost:5000/waiter:latest

docker build -t hello:latest ./hello-service/
docker tag hello:latest localhost:5000/hello:latest
docker push localhost:5000/hello:latest

# 3. Деплой сервисов
echo "Step 4: Deploying services"
kubectl apply -f waiter-service/k8s/service.yml
kubectl apply -f hello-service/k8s/service.yml

# 4. Деплой обработчиков метрик
kubectl apply -f queue-proxy/k8s/deployment.yml
kubectl apply -f billing-agent/k8s/deployment.yml
kubectl apply -f saver/k8s/deployment.yml

echo "Step 5: Running backend services"
docker-compose up -d

# 5. Ожидание готовности
echo "Step 6: Waiting for services to be ready"
kubectl wait ksvc --all --timeout=300s --for=condition=Ready

echo "=== Deployment Complete ==="
echo "Waiter service: curl -H 'Host: waiter.default.knative.demo.com' http://localhost/invoke"
echo "Hello service: curl -H 'Host: hello.default.knative.demo.com' http://localhost/"
echo "Backend API: curl http://localhost:8080/api/v1/health"
