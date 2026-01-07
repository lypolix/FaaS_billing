package handlers

import (
	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/services"
)

type Handler struct {
	BillingService *services.BillingService
	MetricsService *services.MetricsService
}

func NewHandler() Handler {
	return Handler{
		BillingService: services.NewBillingService(database.DB),
		MetricsService: services.NewMetricsService(database.DB),
	}
}
