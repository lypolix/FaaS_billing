package services

import (
	"context"
	"errors"
	"time"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

type MetricsService struct{}

func NewMetricsService() MetricsService { return MetricsService{} }

func (m MetricsService) IngestMetrics(batch []models.UsageRaw) error {
	if len(batch) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	return database.DB.WithContext(ctx).Create(&batch).Error
}
