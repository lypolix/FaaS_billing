-- Включаем UUID расширение
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создаём таблицу системных логов (нужна для reset_free_tier)
CREATE TABLE IF NOT EXISTS public.system_logs (
    id SERIAL PRIMARY KEY,
    message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Функция для обнуления free tier каждый месяц
CREATE OR REPLACE FUNCTION public.reset_free_tier()
RETURNS void AS $$
BEGIN
    INSERT INTO public.system_logs (message, created_at)
    VALUES ('Free tier reset completed', NOW());
END;
$$ LANGUAGE plpgsql;

-- Индексы создаём только если таблицы уже существуют
DO $$
BEGIN
    -- usage_raws
    IF to_regclass('public.usage_raws') IS NOT NULL THEN
        CREATE INDEX IF NOT EXISTS idx_usage_raw_tenant_timestamp
            ON public.usage_raws (tenant_id, timestamp);

        CREATE INDEX IF NOT EXISTS idx_usage_raw_service_timestamp
            ON public.usage_raws (service_id, timestamp);

        CREATE INDEX IF NOT EXISTS idx_usage_raw_metric_name
            ON public.usage_raws (metric_name);
    END IF;

    -- usage_aggregates
    IF to_regclass('public.usage_aggregates') IS NOT NULL THEN
        CREATE INDEX IF NOT EXISTS idx_usage_aggregates_tenant_window
            ON public.usage_aggregates (tenant_id, window_start, window_end);

        CREATE INDEX IF NOT EXISTS idx_usage_aggregates_service_window
            ON public.usage_aggregates (service_id, window_start, window_end);
    END IF;
END
$$;
