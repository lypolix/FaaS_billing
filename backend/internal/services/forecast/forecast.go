package forecast

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

type ForecastRequest struct {
	TenantID  string  `json:"tenant_id"`
	ServiceID *string `json:"service_id"`
	Period    string  `json:"period"` // "1h", "1d", "1w"
}

type ForecastResponse struct {
	ForecastedCost float64            `json:"forecasted_cost"`
	Components     map[string]float64 `json:"components"`
}

func ForecastCost(req ForecastRequest) (*ForecastResponse, error) {
	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, err
	}

	var serviceUUID *uuid.UUID
	if req.ServiceID != nil && *req.ServiceID != "" {
		sid, err := uuid.Parse(*req.ServiceID)
		if err != nil {
			return nil, err
		}
		serviceUUID = &sid
	}

	days, err := periodToDays(req.Period)
	if err != nil {
		return nil, err
	}

	// история за последние 30 дней
	end := time.Now()
	start := end.Add(-30 * 24 * time.Hour)

	var aggs []models.UsageAggregate
	q := database.DB.
		Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", tenantUUID, start, end)

	if serviceUUID != nil {
		q = q.Where("service_id = ?", *serviceUUID)
	}

	if err := q.Find(&aggs).Error; err != nil {
		return nil, err
	}
	if len(aggs) == 0 {
		return &ForecastResponse{
			ForecastedCost: 0,
			Components: map[string]float64{
				"invocations":  0,
				"gb_hours":     0,
				"cold_starts":  0,
				"egress_gb":    0,
			},
		}, nil
	}

	var totalInvocations int64
	var totalMBHours float64
	var totalColdStarts int64
	var totalEgressBytes int64

	for _, a := range aggs {
		totalInvocations += a.Invocations
		totalMBHours += a.TotalMemoryMBHours
		totalColdStarts += int64(a.ColdStarts)
		totalEgressBytes += a.EgressBytes
	}

	avgDailyInvocations := float64(totalInvocations) / 30.0
	avgDailyMBHours := totalMBHours / 30.0
	avgDailyColdStarts := float64(totalColdStarts) / 30.0
	avgDailyEgressGB := (float64(totalEgressBytes) / (1024.0 * 1024.0 * 1024.0)) / 30.0

	forecastInvocations := avgDailyInvocations * float64(days)
	forecastMBHours := avgDailyMBHours * float64(days)
	forecastGBHours := forecastMBHours / 1024.0 // MB*h -> GB*h
	forecastColdStarts := avgDailyColdStarts * float64(days)
	forecastEgressGB := avgDailyEgressGB * float64(days)

	var pricing models.PricingPlan
	// лучше выбирать план по tenant_id (если есть), иначе общий (tenant_id IS NULL)
	err = database.DB.
		Where("tenant_id = ? AND active = true", tenantUUID).
		Or("tenant_id IS NULL AND active = true").
		Order("tenant_id DESC"). // tenant-specific будет "выше"
		First(&pricing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	costInvocations := (forecastInvocations / 1_000_000.0) * pricing.PricePerMillionInvocations
	costGBHours := forecastGBHours * pricing.PricePerGBHour
	costColdStarts := forecastColdStarts * pricing.PricePerColdStart
	costEgress := forecastEgressGB * pricing.PricePerGBEgress

	totalCost := costInvocations + costGBHours + costColdStarts + costEgress

	return &ForecastResponse{
		ForecastedCost: totalCost,
		Components: map[string]float64{
			"invocations": costInvocations,
			"gb_hours":    costGBHours,
			"cold_starts": costColdStarts,
			"egress_gb":   costEgress,
		},
	}, nil
}

func periodToDays(period string) (int, error) {
	switch period {
	case "1h":
		// Нельзя 1/24 как int: получится 0 из-за целочисленного деления. [web:106]
		return 1, nil // минимально 1 день для простого прогноза по дневной агрегации
	case "1d":
		return 1, nil
	case "1w":
		return 7, nil
	default:
		return 0, errors.New("unsupported period, use: 1h, 1d, 1w")
	}
}
