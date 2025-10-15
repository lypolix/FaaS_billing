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
		getEnv("DB_USER", "billinguser"),
		getEnv("DB_PASSWORD", "billingpass"),
		getEnv("DB_NAME", "faasbilling"),
		getEnv("DB_PORT", "5432"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	log.Println("Connected to PostgreSQL")
}

func Migrate() {
	if err := DB.AutoMigrate(
		&models.Tenant{},
		&models.Service{},
		&models.Revision{},
		&models.PricingPlan{},
		&models.UsageRaw{},
		&models.UsageAggregate{},
		&models.Bill{},
	); err != nil {
		log.Fatal("Failed to migrate: ", err)
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
