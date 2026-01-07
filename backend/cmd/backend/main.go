package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/handlers"
)

func main() {
	// подключаемся к БД и делаем миграции
	database.Connect()
	database.Migrate()

	r := gin.Default()
	r.MaxMultipartMemory = 200 << 20 


	h := handlers.NewHandler()

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

		// tenants
		api.POST("/tenants", h.CreateTenant)
		api.GET("/tenants", h.GetTenants)
		api.GET("/tenants/:id", h.GetTenant)

		// services
		api.POST("/services", h.CreateService)
		api.GET("/services", h.GetServices)
		api.POST("/services/:id/upload", h.UploadServiceArtifact)
		api.GET("/artifacts/:service_id/:filename", h.DownloadArtifact)


		// usage aggregates
		api.GET("/usage-aggregates", h.GetUsageAggregates)

		// metrics ingest/aggregate
		api.POST("/metrics/ingest", h.IngestMetrics)
		api.POST("/metrics/aggregate", h.AggregateMetrics)

		// billing
		api.POST("/billing/calculate", h.CalculateCost)
		api.POST("/billing/generate", h.GenerateBill)

		// ml (прокси)
		api.POST("/forecast/cost", h.ProxyForecast)
	}

	addr := os.Getenv("BACKEND_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Println("Backend listening on", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
