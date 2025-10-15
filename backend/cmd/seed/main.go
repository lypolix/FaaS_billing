package main

import (
	"log"
	"time"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

func main() {
	database.Connect()
	database.Migrate()

	// Create tenant
	tenant := models.Tenant{
		Name:     "Demo Tenant",
		Currency: "RUB",
		TaxRate:  0.20,
		Status:   "active",
	}
	if err := database.DB.Create(&tenant).Error; err != nil {
		log.Fatal("create tenant:", err)
	}

	// Create pricing plan
	pricingPlan := models.PricingPlan{
		TenantID:            tenant.ID,
		Name:                "Standard Plan",
		Description:         "Standard pricing for FaaS",
		Currency:            "RUB",
		EffectiveFrom:       time.Now().AddDate(0, -1, 0),
		PricePerInvocation:  0.0000035,
		PricePerMBMs:        0.000016,
		PricePerExecMs:      0.0,
		PricePerColdStart:   0.001,
		FreeTierInvocations: 1000000,
		FreeTierMBMs:        400000,
	}
	if err := database.DB.Create(&pricingPlan).Error; err != nil {
		log.Fatal("create pricing plan:", err)
	}

	// Create service
	service := models.Service{
		TenantID:          tenant.ID,
		Name:              "waiter",
		Namespace:         "default",
		AutoscalingMetric: "concurrency",
		AutoscalingTarget: 10,
		MinScale:          0,
		MaxScale:          10,
	}
	if err := database.DB.Create(&service).Error; err != nil {
		log.Fatal("create service:", err)
	}

	// Create revision
	revision := models.Revision{
		ServiceID:         service.ID,
		RevisionName:      "waiter-00001",
		Image:             "waiter:latest",
		ResourcesMemoryMB: 128,
		ResourcesCPUCores: 0.1,
		ScaleToZero:       true,
	}
	if err := database.DB.Create(&revision).Error; err != nil {
		log.Fatal("create revision:", err)
	}

	log.Println("Demo data created successfully!")
	log.Printf("Tenant ID: %s", tenant.ID)
	log.Printf("Service ID: %s", service.ID)
	log.Printf("Revision ID: %s", revision.ID)
}
