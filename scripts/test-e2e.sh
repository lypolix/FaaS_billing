#!/bin/bash
set -euo pipefail

echo "=== End-to-End Testing ==="

# 1. Тестирование функций
echo "Testing waiter service..."
curl -H "Host: waiter.default.knative.demo.com" "http://localhost/invoke?sleep_ms=100&mem_mb=50"

echo "Testing hello service..."
curl -H "Host: hello.default.knative.demo.com" "http://localhost/"

# 2. Проверка метрик
echo "Checking metrics..."
curl -H "Host: waiter.default.knative.demo.com" "http://localhost/metrics" | grep waiter_requests_total

# 3. Тестирование биллинга
echo "Testing billing calculation..."
curl -X POST http://localhost:8080/api/v1/billing/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "demo-tenant",
    "period_start": "'$(date -u -d '1 hour ago' '+%Y-%m-%dT%H:%M:%SZ')'",
    "period_end": "'$(date -u '+%Y-%m-%dT%H:%M:%SZ')'"
  }'

echo "=== Tests Complete ==="
