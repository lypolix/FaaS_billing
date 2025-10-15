#!/bin/bash
set -euo pipefail

echo "Installing Prometheus with Helm..."
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
helm repo update

# Установка с кастомными значениями для скрейпа Knative
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
  --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
  --namespace monitoring --create-namespace

echo "Prometheus UI available at: kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090"
