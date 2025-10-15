package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/handlers"
)

func main() {
	database.Connect()
	database.Migrate()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	h := handlers.NewHandler()

	api := r.Group("/api/v1")
	{
		api.POST("/tenants", h.CreateTenant)
		api.GET("/tenants", h.GetTenants)
		api.GET("/tenants/:id", h.GetTenant)

		api.POST("/services", h.CreateService)
		api.GET("/services", h.GetServices)

		api.GET("/usage-aggregates", h.GetUsageAggregates)
		api.POST("/metrics/ingest", h.IngestMetrics)

		api.POST("/billing/calculate", h.CalculateCost)
		api.POST("/billing/generate", h.GenerateBill)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	log.Println("Starting server on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}
