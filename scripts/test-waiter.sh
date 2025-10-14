#!/bin/bash

WAITER_URL="http://waiter.default.knative.demo.com"

echo "🧪 Testing waiter service..."

# Тест 1: Базовый вызов
echo "Test 1: Basic invocation"
curl -H "Host: waiter.default.knative.demo.com" \
     "http://localhost/invoke?sleep_ms=200&mem_mb=50" | jq .

# Тест 2: CPU нагрузка  
echo "Test 2: CPU intensive"
curl -H "Host: waiter.default.knative.demo.com" \
     "http://localhost/invoke?cpu_spin_ms=1000&mem_mb=100" | jq .

# Тест 3: Memory тест
echo "Test 3: Memory allocation"  
curl -H "Host: waiter.default.knative.demo.com" \
     "http://localhost/invoke?mem_mb=200&sleep_ms=500" | jq .

# Тест 4: Проверка метрик
echo "Test 4: Metrics endpoint"
curl -H "Host: waiter.default.knative.demo.com" \
     "http://localhost/metrics" | head -20

echo "✅ Testing completed"
