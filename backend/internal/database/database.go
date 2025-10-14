package database

import (
    "fmt"
    "log"
    "os"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    
    "github.com/lypolix/FaaS-billing/internal/models"
)

var DB *gorm.DB

func Connect() {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
        getEnv("DB_HOST", "localhost"),
        getEnv("DB_USER", "billing_user"),
        getEnv("DB_PASSWORD", "billing_pass"),
        getEnv("DB_NAME", "faas_billing"),
        getEnv("DB_PORT", "5432"),
    )
    
    var err error
    DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    log.Println("Connected to PostgreSQL database")
}

func Migrate() {
    log.Println("Starting database migration...")
    
    err := DB.AutoMigrate(
        &models.Tenant{},
        &models.Service{},
        &models.Revision{},
        &models.PricingPlan{},
        &models.UsageRaw{},
        &models.UsageAggregate{},
        &models.Bill{},
        &models.LineItem{},
    )
    
    if err != nil {
        log.Fatal("Failed to migrate database:", err)
    }
    
    log.Println("Database migration completed")
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
