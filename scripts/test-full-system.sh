#!/bin/bash
set -euo pipefail

echo "=== FaaS Billing System Full Test ==="

# Проверяем что все сервисы запущены
echo "[1] Checking services..."
curl -s http://localhost:8080/api/v1/health || { echo "Backend не доступен"; exit 1; }
curl -s http://localhost:8081/healthz || { echo "Queue-proxy не доступен"; exit 1; }
curl -s http://localhost:8082/health || { echo "AI-forecast не доступен"; exit 1; }

# Генерируем нагрузку на waiter
echo "[2] Generating load on waiter function..."
for i in {1..50}; do
    curl -s -H "Host: waiter.default.knative.demo.com" \
        "http://localhost/invoke?sleep_ms=100&mem_mb=64&egress=true" > /dev/null &
done
wait

echo "[3] Waiting for metrics collection..."
sleep 30

# Проверяем что метрики собираются
echo "[4] Checking Prometheus metrics..."
curl -s -H "Host: waiter.default.knative.demo.com" "http://localhost/metrics" | grep waiter_requests_total

# Запускаем агрегацию
echo "[5] Running aggregation..."
curl -s -X POST http://localhost:8080/api/v1/metrics/aggregate \
    -H "Content-Type: application/json" \
    -d "{
        \"start_time\": \"$(date -u -d '1 hour ago' '+%Y-%m-%dT%H:%M:%SZ')\",
        \"end_time\": \"$(date -u '+%Y-%m-%dT%H:%M:%SZ')\",
        \"window_size\": \"5m\"
    }"

# Рассчитываем биллинг
echo "[6] Calculating billing..."
BILLING_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/billing/calculate \
    -H "Content-Type: application/json" \
    -d "{
        \"tenant_id\": \"demo-tenant\",
        \"start_time\": \"$(date -u -d '1 hour ago' '+%Y-%m-%dT%H:%M:%SZ')\",
        \"end_time\": \"$(date -u '+%Y-%m-%dT%H:%M:%SZ')\"
    }")

echo "Billing result:"
echo "$BILLING_RESULT" | jq '.'

# Проверяем прогноз
echo "[7] Testing ML forecast..."
FORECAST_RESULT=$(curl -s -X POST http://localhost:8082/forecast/cost \
    -H "Content-Type: application/json" \
    -d "{
        \"tenant_id\": \"demo-tenant\",
        \"period\": \"1d\"
    }")

echo "Forecast result:"
echo "$FORECAST_RESULT" | jq '.'

# Проверяем веб-интерфейс
echo "[8] Checking web interface..."
curl -s http://localhost:3000 | grep -q "FaaS Billing" && echo "Frontend OK" || echo "Frontend issue"

echo "=== Full System Test Complete ==="
echo "Система готова к использованию!"
echo "Frontend: http://localhost:3000"
echo "Backend API: http://localhost:8080"
echo "Prometheus: kubectl port-forward svc/prometheus-server 9090:80"
echo "Grafana: kubectl port-forward svc/grafana 3001:80"
