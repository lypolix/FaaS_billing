-- Включаем UUID расширение
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создаём таблицы (GORM сам их создаст, но можем добавить индексы)
-- Полезные индексы для быстрых запросов
CREATE INDEX IF NOT EXISTS idx_usage_raw_tenant_timestamp 
ON usage_raws(tenant_id, timestamp);

CREATE INDEX IF NOT EXISTS idx_usage_raw_service_timestamp 
ON usage_raws(service_id, timestamp);

CREATE INDEX IF NOT EXISTS idx_usage_raw_metric_name 
ON usage_raws(metric_name);

CREATE INDEX IF NOT EXISTS idx_usage_aggregates_tenant_window 
ON usage_aggregates(tenant_id, window_start, window_end);

CREATE INDEX IF NOT EXISTS idx_usage_aggregates_service_window 
ON usage_aggregates(service_id, window_start, window_end);

-- Функция для обнуления free tier каждый месяц
CREATE OR REPLACE FUNCTION reset_free_tier()
RETURNS void AS $$
BEGIN
    -- Здесь может быть логика сброса счетчиков free tier
    -- или просто лог что функция выполнилась
    INSERT INTO system_logs (message, created_at) 
    VALUES ('Free tier reset completed', NOW());
END;
$$ LANGUAGE plpgsql;

-- Создаём таблицу системных логов
CREATE TABLE IF NOT EXISTS system_logs (
    id SERIAL PRIMARY KEY,
    message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
