package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

func main() {
	database.Connect()
	database.Migrate()

	tenant := models.Tenant{
		ID:           uuid.New(),
		Name:         "Demo Tenant",
		BillingEmail: "billing@example.com",
		Currency:     "RUB",
		Timezone:     "Europe/Moscow",
	}
	_ = database.DB.Create(&tenant).Error

	service := models.Service{
		ID:            uuid.New(),
		TenantID:      tenant.ID,
		Name:          "waiter",
		Namespace:     "default",
		Runtime:       "go",
		MemoryLimitMB: 256,
		CPULimitCores: 0.5,
	}
	_ = database.DB.Create(&service).Error

	plan := models.PricingPlan{
		Name:                      "Yandex Default",
		Currency:                  "RUB",
		PricePerMillionInvocations: 17.28,
		PricePerGBHour:            5.9076,
		PricePerColdStart:         0,
		PricePerGBEgress:          1.6524,
		PricePerGBHourProvisioned: 1.296,
		PricePerGBHourActive:      2.484,
		FreeTierInvocations:       1_000_000,
		FreeTierGBHours:           10.0,
		FreeTierEgressGB:          100.0,
		Active:                    true,
		CreatedAt:                 time.Now(),
	}
	_ = database.DB.Create(&plan).Error

	log.Println("Seed completed")
}
