package main

import (
    "log"
    
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    
    "github.com/lypolix/FaaS-billing/internal/database"
    "github.com/lypolix/FaaS-billing/internal/handlers"
)

func main() {
    // Connect to database
    database.Connect()
    database.Migrate()
    
    // Initialize Gin
    r := gin.Default()
    
    // CORS middleware
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
    }))
    
    // Initialize handlers
    handler := handlers.NewHandler()
    
    // API routes
    api := r.Group("/api/v1")
    {
        // Tenants
        api.POST("/tenants", handler.CreateTenant)
        api.GET("/tenants", handler.GetTenants)
        api.GET("/tenants/:id", handler.GetTenant)
        
        // Services
        api.POST("/services", handler.CreateService)
        api.GET("/services", handler.GetServices)
        
        // Usage
        api.GET("/usage-aggregates", handler.GetUsageAggregates)
        api.POST("/metrics/ingest", handler.IngestMetrics)
        
        // Billing
        api.POST("/billing/calculate", handler.CalculateCost)
        api.POST("/billing/generate", handler.GenerateBill)

		api.POST("/metrics/ingest", handler.IngestMetrics)
    }
    
    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })
    
    log.Println("Starting server on :8081")
    r.Run(":8081")
}
