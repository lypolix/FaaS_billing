package services

import (
	"context"
	"errors"
	"time"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

type MetricsService struct {
	// при необходимости: клиенты OTLP/Prometheus, конфиг и т.д.
}

func NewMetricsService() *MetricsService {
	return &MetricsService{}
}

// IngestMetrics — базовая реализация приёма сырых метрик.
// На вход подаются нормализованные записи (уже с unit/temporality/labels).
// В проде это может быть OTLP приемник, преобразователь Prometheus → UsageRaw и т.п.
func (m *MetricsService) IngestMetrics(batch []models.UsageRaw) error {
	if len(batch) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Простая валидация
	for i := range batch {
		if batch[i].Timestamp.IsZero() || batch[i].Metric == "" {
			return errors.New("invalid metric row: empty timestamp or metric")
		}
		if batch[i].Unit == "" {
			batch[i].Unit = "count"
		}
		if batch[i].Temporality == "" {
			batch[i].Temporality = "delta"
		}
	}

	// Bulk insert
	return database.DB.WithContext(ctx).Create(&batch).Error
}
