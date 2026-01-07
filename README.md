# FaaS Billing — Система биллинга для serverless-функций на Knative

FaaS Billing — система биллинга и аналитики для serverless‑функций в Kubernetes/Knative, которая собирает сырые метрики исполнения, агрегирует их по временным окнам и рассчитывает стоимость использования по тарифным планам с возможностью экспорта счетов и прогнозирования будущих затрат.


---

## Быстрый старт

### Требования
- Docker Desktop с включённым Kubernetes (context: `docker-desktop`)
- kubectl, docker, helm
- Порт 5000 свободен (локальный реестр)

### Запуск 
1) Установить Knative + Kourier, настроить домен, развернуть базовые сервисы:
```
chmod +x scripts/setup-k8s.sh scripts/setup-prometheus.sh scripts/deploy-all.sh
./scripts/setup-k8s.sh
```

2) Установить Prometheus/Grafana стек
```
./scripts/setup-prometheus.sh
```

3) Собрать и развернуть все микросервисы
```
./scripts/deploy-all.sh
```

4) Запустить локальные зависимости (PostgreSQL, Redis, Backend)
```
docker-compose up -d
```

5) End-to-end тест

```
./scripts/test-e2e.sh
```

### Проверка работоспособности

Проверить статус всех сервисов
```
kubectl get ksvc,deploy,svc -A | grep -E "(waiter|queue|saver|billing)"
```

Протестировать функцию waiter

```
curl -H "Host: waiter.default.knative.demo.com"
"http://localhost/invoke?sleep_ms=200&mem_mb=50"
```

Проверить метрики

```
curl -H "Host: waiter.default.knative.demo.com" "http://localhost/metrics"
```

Тест биллинга

```
curl -X POST http://localhost:8080/api/v1/billing/calculate
-H "Content-Type: application/json"
-d '{"tenant_id":"demo","period_start":"2025-10-15T00:00:00Z","period_end":"2025-10-15T23:59:59Z"}'
```

---

## Архитектура решения

### Общая идея
Система собирает детальные метрики выполнения serverless-функций (количество вызовов, время выполнения, потребляемая память, холодные старты), агрегирует их в временные окна и рассчитывает стоимость по гибким тарифным планам с поддержкой ML-прогнозирования и визуализацией.

### Компоненты (по схеме)

| Компонент | Назначение | Папка | Технологии |
|-----------|------------|-------|-----------|
| **user-container** | Serverless-функции пользователя | `waiter-service/`, `hello-service/`, `echo-service/` | Go/Gin, Python, Knative |
| **queue-proxy** | Сбор и буферизация событий метрик | `queue-proxy/` | Go/Gin + Redis |
| **billing-agent** | Обработка метрик, детекция холодных стартов | `billing-agent/` | Go + cgroups/RAM monitoring |
| **saver** | Сохранение данных из Redis в PostgreSQL | `saver/` | Go + GORM |
| **billing-API** | REST API для расчётов, отчётов, тарифов | `backend/` | Go/Gin + PostgreSQL |
| **PostgreSQL** | Долговременное хранение метрик и биллинга | `deployment/postgres/` | PostgreSQL 15 |
| **Redis** | Кеш и очередь событий | `deployment/redis/` | Redis 7 |
| **Frontend** | Веб-интерфейс дашбордов и отчётов | `frontend/` (не в дереве) | React/Next.js |
| **AI модуль** | ML-прогнозы расходов и оптимизации | `ml/` | Python/scikit-learn |

---

## Модели данных

### Основные таблицы (PostgreSQL)

#### Tenants (Арендаторы)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Уникальный идентификатор |
| name | VARCHAR(255) | Название организации |
| created_at | TIMESTAMP | Дата создания |
| updated_at | TIMESTAMP | Дата обновления |
| billing_email | VARCHAR(255) | Email для счетов |
| currency | VARCHAR(3) | Валюта (RUB, USD) |
| timezone | VARCHAR(50) | Временная зона |

#### Services (Сервисы/Функции)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Уникальный идентификатор |
| tenant_id | UUID | FK на Tenants |
| name | VARCHAR(255) | Имя сервиса |
| namespace | VARCHAR(255) | K8s namespace |
| created_at | TIMESTAMP | Дата создания |
| runtime | VARCHAR(50) | go, python, nodejs |
| memory_limit_mb | INTEGER | Лимит памяти |
| cpu_limit_cores | DECIMAL(3,2) | Лимит CPU |

#### Revisions (Ревизии сервисов)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Уникальный идентификатор |
| service_id | UUID | FK на Services |
| name | VARCHAR(255) | Имя ревизии (waiter-00001) |
| image | VARCHAR(500) | Docker образ |
| created_at | TIMESTAMP | Дата создания |
| config_env | JSONB | Переменные окружения |
| scaling_config | JSONB | Настройки автоскейлинга |

#### UsageRaw (Сырые метрики)
| Поле | Тип | Описание |
|------|-----|----------|
| id | BIGSERIAL | Уникальный идентификатор |
| timestamp | TIMESTAMP | Время события |
| tenant_id | UUID | FK на Tenants |
| service_id | UUID | FK на Services |
| revision_id | UUID | FK на Revisions |
| metric_name | VARCHAR(100) | invocations, duration_ms, memory_mb |
| value | DECIMAL(15,6) | Значение метрики |
| labels | JSONB | Дополнительные метки |
| request_id | VARCHAR(255) | ID запроса (трейсинг) |

#### UsageAggregate (Агрегированные метрики)
| Поле | Тип | Описание |
|------|-----|----------|
| id | BIGSERIAL | Уникальный идентификатор |
| window_start | TIMESTAMP | Начало окна агрегации |
| window_end | TIMESTAMP | Конец окна агрегации |
| window_size | VARCHAR(10) | 5m, 1h, 1d |
| tenant_id | UUID | FK на Tenants |
| service_id | UUID | FK на Services |
| revision_id | UUID | FK на Revisions |
| invocations | BIGINT | Количество вызовов |
| total_duration_ms | BIGINT | Общее время выполнения |
| avg_duration_ms | DECIMAL(10,3) | Среднее время выполнения |
| p50_duration_ms | DECIMAL(10,3) | Медиана времени |
| p95_duration_ms | DECIMAL(10,3) | 95-й процентиль |
| max_memory_mb | DECIMAL(10,3) | Пиковое потребление памяти |
| avg_memory_mb | DECIMAL(10,3) | Среднее потребление |
| cold_starts | INTEGER | Количество холодных стартов |
| errors | INTEGER | Количество ошибок |

#### PricingPlans (Тарифные планы)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Уникальный идентификатор |
| name | VARCHAR(255) | Название тарифа |
| tenant_id | UUID | FK на Tenants (NULL = общий) |
| currency | VARCHAR(3) | Валюта |
| price_per_invocation | DECIMAL(10,6) | Цена за вызов |
| price_per_mb_ms | DECIMAL(10,6) | Цена за МБ×мс |
| price_per_cold_start | DECIMAL(10,6) | Цена за холодный старт |
| price_per_cpu_ms | DECIMAL(10,6) | Цена за мс CPU |
| free_tier_invocations | BIGINT | Бесплатные вызовы/месяц |
| free_tier_mb_ms | BIGINT | Бесплатные МБ×мс/месяц |
| created_at | TIMESTAMP | Дата создания |
| active | BOOLEAN | Активен ли тариф |

#### Bills (Счета)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Уникальный идентификатор |
| tenant_id | UUID | FK на Tenants |
| period_start | TIMESTAMP | Начало периода |
| period_end | TIMESTAMP | Конец периода |
| total_amount | DECIMAL(15,2) | Общая сумма |
| currency | VARCHAR(3) | Валюта |
| status | VARCHAR(50) | draft, final, paid |
| created_at | TIMESTAMP | Дата создания |
| line_items | JSONB | Детализация по статьям |

#### ML_Predictions (ML-прогнозы)
| Поле | Тип | Описание |
|------|-----|----------|
| id | UUID | Уникальный идентификатор |
| tenant_id | UUID | FK на Tenants |
| service_id | UUID | FK на Services (NULL = общий) |
| prediction_date | DATE | Дата прогноза |
| horizon_days | INTEGER | Горизонт (1, 7, 30 дней) |
| predicted_invocations | BIGINT | Прогноз вызовов |
| predicted_cost | DECIMAL(15,2) | Прогноз стоимости |
| confidence | DECIMAL(5,4) | Уверенность модели (0-1) |
| model_version | VARCHAR(50) | Версия модели |
| created_at | TIMESTAMP | Дата создания |

---

## Текущие микросервисы и эндпоинты

### waiter-service (Тестовая функция)
**Локация**: `waiter-service/`  
**Технологии**: Go/Gin, Knative Service  
**Назначение**: Эталонная функция для тестирования автоскейлинга и сбора метрик

#### Эндпоинты:
| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/invoke` | Эмуляция нагрузки: `?sleep_ms=200&mem_mb=50&cpu_spin_ms=100` |
| GET | `/metrics` | Prometheus-метрики |
| GET | `/healthz` | Health check |
| GET | `/readiness` | Readiness probe |

#### Собираемые метрики:
- `waiter_requests_total{method,endpoint,status}` — счётчик запросов
- `waiter_request_duration_seconds` — гистограмма длительности
- `waiter_memory_usage_bytes` — текущее потребление памяти
- `waiter_cold_starts_total` — счётчик холодных стартов

### queue-proxy (Буферизация событий)
**Локация**: `queue-proxy/`  
**Технологии**: Go/Gin + Redis  
**Назначение**: Приём и буферизация событий метрик

#### Эндпоинты:
| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/metrics/collect` | Приём событий метрик (одиночно/батчом) |
| POST | `/metrics/pop` | Извлечение события для обработки |
| GET | `/metrics` | Внутренние метрики сервиса |
| GET | `/healthz` | Health check |

#### Формат событий:
```
{
"timestamp": "2025-10-15T12:00:00Z",
"tenant_id": "demo-tenant",
"service_name": "waiter",
"revision": "waiter-00001",
"invocations": 1,
"duration_seconds": 0.234,
"memory_mb": 64.5,
"cold_start": false,
"labels": {"method": "GET", "status": "200"}
}
```

### saver (Персистентность)

**Локация**: `saver/`  
**Технологии**: Go + GORM + PostgreSQL  
**Назначение**: Сохранение событий из Redis в PostgreSQL

#### Функции:
- Читает события из Redis очереди (`metrics_queue`)
- Нормализует и сохраняет в таблицу `UsageRaw`
- Логирует ошибки и метрики производительности
- Поддерживает батчевую вставку для высокой нагрузки

### billing-agent (Обработчик метрик)
**Локация**: `billing-agent/`  
**Технологии**: Go + системный мониторинг  
**Назначение**: Расширенная обработка метрик, детекция аномалий

#### Функции:
- Мониторинг ресурсов через cgroups
- Детекция холодных стартов по времени первого запроса
- Расчёт дополнительных метрик (p50, p95, коэффициенты)
- Отправка агрегированных данных в billing-API

### backend (Billing API)
**Локация**: `backend/`  
**Технологии**: Go/Gin + PostgreSQL + GORM  
**Назначение**: Центральный API для биллинга, отчётов и управления
#### Готовые эндпоинты (текущий прогресс)
| Метод | Путь | Описание | Статус |
|------:|------|----------|:------:|
| GET | `/api/v1/health` | Health check | Готов |
| POST | `/api/v1/tenants` | Создание арендатора | Готов |
| GET | `/api/v1/tenants` | Список арендаторов | Готов |
| GET | `/api/v1/tenants/:id` | Детали арендатора | Готов |
| POST | `/api/v1/services` | Регистрация сервиса | Готов |
| GET | `/api/v1/services` | Список сервисов | Готов |
| POST | `/api/v1/services/:id/upload` | Загрузка артефакта (файла) сервиса | Готов |
| GET | `/api/v1/artifacts/:service_id/:filename` | Скачать артефакт сервиса | Готов |
| GET | `/api/v1/usage-aggregates` | Получить агрегированные метрики (фильтры: `tenant_id`, `service_id`, `start_time`, `end_time`) | Готов |
| POST | `/api/v1/metrics/ingest` | Приём сырых метрик | Готов |
| POST | `/api/v1/metrics/aggregate` | Ручной запуск агрегации | Готов |
| POST | `/api/v1/billing/calculate` | Расчёт стоимости (без сохранения счёта) | Готов |
| POST | `/api/v1/billing/generate` | Расчёт + сохранение счёта (draft) | Готов |
| POST | `/api/v1/forecast/cost` | Прокси в ML-сервис прогноза | Готов |
| GET | `/api/v1/pricing-plans` | Список тарифных планов |  Готов |
| PUT | `/api/v1/tenants/:id/pricing-plan` | Назначить тарифный план тенанту (вариант B: явный `pricing_plan_id`) |  Готов |
| GET | `/api/v1/tenants/:id/pricing-plan` | Получить текущий тарифный план тенанта |  Готов |

#### Планируется (ещё нет реализации)
| Метод | Путь | Описание | Статус |
|------:|------|----------|:------:|
| GET | `/api/v1/billing/reports/:tenant_id` | Отчёты/история счетов по арендатору |  Планируется |
| POST | `/api/v1/pricing/plans` | Управление тарифами (CRUD) |  Планируется |
| GET | `/api/v1/usage/dashboard/:tenant_id` | Дашборд метрик (агрегации + графики) | Планируется |


### hello-service & echo-service (Дополнительные функции)
**Локация**: `hello-service/`, `echo-service/`  
**Технологии**: Python, ealen/echo-server  
**Назначение**: Дополнительные тестовые функции для демонстрации

---

## Инфраструктура и хранилища

### PostgreSQL (Основная БД)
**Локация**: `deployment/postgres/init.sql`  
**Назначение**: Долговременное хранение всех данных системы
- Схема автоматически создаётся при первом запуске
- Поддержка временных индексов для быстрой агрегации
- Настроены foreign keys и constraints для целостности данных

### Redis (Кеш и очереди)
**Локация**: `deployment/redis/redis.yml`  
**Назначение**: 
- Очередь событий метрик (`metrics_queue`)
- Кеш часто используемых данных (тарифы, конфигурации)
- Session storage для веб-интерфейса

### Prometheus + Grafana (Мониторинг)
**Локация**: `deployment/prometheus/`, `deployment/grafana/`  
**Настройки**:
- Автодискавери Knative Services по аннотациям
- Предустановленные дашборды для метрик функций
- Алерты на аномальное потребление ресурсов

---

## Автоскейлинг и Kubernetes

### Как работает автоскейлинг Knative:
1. **Scale-to-zero**: При отсутствии трафика поды функций останавливаются
2. **Cold start**: Первый запрос после простоя запускает новый под (~1-3 сек)
3. **Concurrency-based scaling**: Масштабирование по количеству одновременных запросов
4. **Custom metrics**: Можно настроить масштабирование по custom метрикам

### Конфигурация автоскейлинга (пример):
```
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
name: waiter
spec:
template:
metadata:
annotations:
autoscaling.knative.dev/minScale: "0" # Scale-to-zero
autoscaling.knative.dev/maxScale: "10" # Максимум подов
autoscaling.knative.dev/target: "100" # Целевая конкурентность
autoscaling.knative.dev/metric: "concurrency" # Метрика масштабирования
autoscaling.knative.dev/class: "kpa.autoscaling.knative.dev"
```

### Загрузка новых функций:
1. **Создать Dockerfile** с вашим приложением на порту 8080
2. **Собрать и запушить образ**:
```
docker build -t myfunction:latest .
docker tag myfunction:latest localhost:5000/myfunction:latest
docker push localhost:5000/myfunction:latest
```

3. **Создать Knative Service манифест**:

```
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
name: myfunction
spec:
template:
metadata:
annotations:
prometheus.io/scrape: "true"
prometheus.io/path: "/metrics"
prometheus.io/port: "8080"
spec:
containers:
- image: localhost:5000/myfunction:latest
ports:
- containerPort: 8080
```

4. **Применить манифест**: 
```
`kubectl apply -f myfunction.yml`
```

---

## Текущие метрики и мониторинг

### Собираемые метрики по функциям:
| Метрика | Тип | Описание |
|---------|-----|----------|
| `*_requests_total` | Counter | Общее количество запросов |
| `*_request_duration_seconds` | Histogram | Распределение времени выполнения |
| `*_memory_usage_bytes` | Gauge | Текущее потребление памяти |
| `*_cold_starts_total` | Counter | Количество холодных стартов |
| `*_errors_total` | Counter | Количество ошибок |

### Системные метрики:
| Сервис | Метрики |
|--------|---------|
| queue-proxy | Пропускная способность, размер очереди, ошибки |
| saver | Скорость записи, задержки БД, failed operations |
| billing-agent | Обработано событий, время обработки |
| backend | HTTP метрики, время ответа API, активные подключения |

---

## Что реализовано:
**Прогноз на 1 час / 1 день / 1 неделю.**

1) Берёт историю из UsageAggregate.
2) Считает стоимость: invocations + memory + cold starts * price.
**Описание переменных:**
Frontend:
- invocations — количество вызовов сервиса.
- memory — потребление памяти (MB×ms).
- cold_starts — количество холодных стартов.
- total_cost — итоговая стоимость.
- line_items — детализация счёта.
ML:
- ForecastRequest:
- tenant_id — арендатор.
- period — "1h", "1d", "1w".
- ForecastResponse:
- forecasted_cost — прогнозируемая стоимость.
- components — детализация (invocations, memory, cold starts)

## План дальнейшей реализации

### Фаза 1: Завершение базового биллинга 

#### Backend эндпоинты:
1. **POST /api/v1/metrics/ingest**
- Приём батчей UsageRaw из saver
- Валидация и дедупликация данных
- Bulk insert в PostgreSQL

2. **POST /api/v1/metrics/aggregate** 
- Агрегация UsageRaw по временным окнам (5m, 1h, 1d)
- Расчёт статистик (avg, p50, p95, max)
- Запись в UsageAggregate

3. **POST /api/v1/billing/calculate**
- Применение тарифных планов

4. **GET/POST /api/v1/pricing/plans**
- CRUD для тарифных планов

5. **Реализовать автоматическую загрузку функций и эндпоинт для её приёма и помощения в кластер**
   
#### Автоматизация:
- Cron job для автоматической агрегации каждые 5 минут
- Cleanup старых данных (retention policy)
- Бэкапы PostgreSQL

### Фаза 2: Веб-интерфейс и дашборды 

#### Frontend (React/Next.js):
1. **Дашборд арендатора**:
- Графики потребления в реальном времени
- Breakdown по сервисам и ревизиям
- Текущие расходы за месяц

2. **Детализированные отчёты**:
- Фильтрация по датам, сервисам, тегам
- Экспорт в CSV/PDF
- Сравнение периодов

3. **Управление тарифами**:
- Настройка цен и free tier
- Симулятор стоимости
- История изменений тарифов

4. **Мониторинг функций**:
- Статус всех функций
- Метрики производительности
- Логи и трейсинг

#### API расширения:
- WebSocket для real-time обновлений
- GraphQL endpoint для гибких запросов
- Webhook уведомления о превышении бюджета

### Фаза 3: ML и аналитика 

#### ML модуль (Python/scikit-learn):
1. **Прогнозирование расходов**:
```
ml/models/cost_prediction.py
Time series forecasting (ARIMA, Prophet)
```

**Сезонность и тренды**

**Доверительные интервалы**


2. **Рекомендации по оптимизации**:

ml/models/optimization.py
1) Анализ паттернов использования

2) Рекомендации по memory/CPU limits

3) Оптимальные настройки автоскейлинга


3. **Детекция аномалий**:
ml/models/anomaly_detection.py

1) Необычное потребление ресурсов

2) Подозрительные паттерны трафика

3) Алерты в реальном времени


#### ML API эндпоинты:
| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/ml/predict` | Прогноз расходов |
| POST | `/api/v1/ml/optimize` | Рекомендации по оптимизации |
| POST | `/api/v1/ml/anomalies` | Детекция аномалий |
| GET | `/api/v1/ml/insights/:tenant_id` | Аналитические инсайты |

#### Фичи ML:
- **Прогноз бюджета**: "При текущих трендах месячные расходы составят X₽"
- **Рекомендации масштабирования**: "Увеличьте minScale до 2 для сервиса A"  
- **Cost optimization**: "Смените тарифный план для экономии 15%"
- **Capacity planning**: "Добавьте ресурсы через 2 недели"

### Фаза 4: Продвинутые фичи

#### Расширенная аналитика:
1. **Multi-tenancy**:
- Изоляция данных между арендаторами
- Общие и персональные тарифы
- Billing по организациям

2. **Advanced pricing**:
- Тарифы по времени суток/дням недели
- Volume discounts
- Reserved capacity pricing
- Spot pricing для неприоритетных задач

3. **Integration & Export**:
- Интеграция с внешними системами учёта
- API для биллинг провайдеров
- Экспорт в 1С, SAP
- Webhook уведомления

4. **Compliance & Security**:
- GDPR compliance для персональных данных
- Audit logs всех операций
- Role-based access control (RBAC)
- Encryption at rest and in transit

---


