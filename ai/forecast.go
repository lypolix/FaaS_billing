package main

import (
	"time"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

type ForecastRequest struct {
	TenantID  string `json:"tenant_id"`
	ServiceID *string `json:"service_id"`
	Period    string `json:"period"` // "1h", "1d", "1w"
}

type ForecastResponse struct {
	ForecastedCost float64            `json:"forecasted_cost"`
	Components     map[string]float64 `json:"components"`
}

// ForecastCost — прогнозирует стоимость на основе истории
func ForecastCost(req ForecastRequest) (*ForecastResponse, error) {
	// Берём историю за последние 30 дней
	end := time.Now()
	start := end.Add(-30 * 24 * time.Hour)

	var aggregates []models.UsageAggregate
	query := database.DB.Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", req.TenantID, start, end)
	if req.ServiceID != nil {
		query = query.Where("service_id = ?", *req.ServiceID)
	}
	query.Find(&aggregates)

	// Простая модель: усреднение + тренд
	var totalInvocations, totalMBMs, totalColdStarts int64
	for _, agg := range aggregates {
		totalInvocations += agg.Invocations
		totalMBMs += agg.MemMBMsSum
		totalColdStarts += agg.ColdStarts
	}

	avgDailyInvocations := float64(totalInvocations) / 30
	avgDailyMBMs := float64(totalMBMs) / 30
	avgDailyColdStarts := float64(totalColdStarts) / 30

	var days int
	switch req.Period {
	case "1h":
		days = 1 / 24
	case "1d":
		days = 1
	case "1w":
		days = 7
	}

	forecastedInvocations := avgDailyInvocations * float64(days)
	forecastedMBMs := avgDailyMBMs * float64(days)
	forecastedColdStarts := avgDailyColdStarts * float64(days)

	// Берём pricing plan
	var pricing models.PricingPlan
	database.DB.First(&pricing)

	forecastedCost := forecastedInvocations * pricing.PricePerInvocation +
		forecastedMBMs * pricing.PricePerMBMs +
		forecastedColdStarts * pricing.PricePerColdStart

	return &ForecastResponse{
		ForecastedCost: forecastedCost,
		Components: map[string]float64{
			"invocations": forecastedInvocations,
			"memory":      forecastedMBMs,
			"cold_starts": forecastedColdStarts,
		},
	}, nil
}
