package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}
	return json.Unmarshal(value.([]byte), j)
}

type Tenant struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name         string    `json:"name" gorm:"not null"`
	BillingEmail string    `json:"billing_email"`
	Currency     string    `json:"currency" gorm:"default:'RUB'"`
	Timezone     string    `json:"timezone" gorm:"default:'UTC'"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Service struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID      uuid.UUID `json:"tenant_id" gorm:"not null"`
	Name          string    `json:"name" gorm:"not null"`
	Namespace     string    `json:"namespace" gorm:"default:'default'"`
	Runtime       string    `json:"runtime"`
	MemoryLimitMB int       `json:"memory_limit_mb"`
	CPULimitCores float64   `json:"cpu_limit_cores"`
	CreatedAt     time.Time `json:"created_at"`
	
	Tenant Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
}

type Revision struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ServiceID     uuid.UUID `json:"service_id" gorm:"not null"`
	Name          string    `json:"name" gorm:"not null"`
	Image         string    `json:"image"`
	ConfigEnv     JSONB     `json:"config_env" gorm:"type:jsonb"`
	ScalingConfig JSONB     `json:"scaling_config" gorm:"type:jsonb"`
	CreatedAt     time.Time `json:"created_at"`
	
	Service Service `json:"service" gorm:"foreignKey:ServiceID"`
}

type UsageRaw struct {
	ID        uint      `json:"id" gorm:"primary_key"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"index"`
	ServiceID uuid.UUID `json:"service_id" gorm:"index"`
	RevisionID *uuid.UUID `json:"revision_id"`
	
	MetricName string  `json:"metric_name"` // "invocations", "duration_ms", "memory_mb", "cold_starts", "egress_bytes"
	Value      float64 `json:"value"`
	Labels     JSONB   `json:"labels" gorm:"type:jsonb"`
	RequestID  string  `json:"request_id"`
	
	Tenant   Tenant   `json:"tenant" gorm:"foreignKey:TenantID"`
	Service  Service  `json:"service" gorm:"foreignKey:ServiceID"`
	Revision *Revision `json:"revision" gorm:"foreignKey:RevisionID"`
}

type UsageAggregate struct {
	ID          uint      `json:"id" gorm:"primary_key"`
	WindowStart time.Time `json:"window_start" gorm:"index"`
	WindowEnd   time.Time `json:"window_end" gorm:"index"`
	WindowSize  string    `json:"window_size"` // "5m", "1h", "1d"
	
	TenantID   uuid.UUID  `json:"tenant_id" gorm:"index"`
	ServiceID  uuid.UUID  `json:"service_id" gorm:"index"`
	RevisionID *uuid.UUID `json:"revision_id"`
	
	// Основные метрики
	Invocations      int64   `json:"invocations"`
	TotalDurationMS  int64   `json:"total_duration_ms"`
	AvgDurationMS    float64 `json:"avg_duration_ms"`
	P50DurationMS    float64 `json:"p50_duration_ms"`
	P95DurationMS    float64 `json:"p95_duration_ms"`
	
	// Память в МБ×час (для тарификации)
	MaxMemoryMB      float64 `json:"max_memory_mb"`
	AvgMemoryMB      float64 `json:"avg_memory_mb"`
	TotalMemoryMBHours float64 `json:"total_memory_mb_hours"` // ключевое для биллинга
	
	// Дополнительные метрики
	ColdStarts       int     `json:"cold_starts"`
	Errors           int     `json:"errors"`
	EgressBytes      int64   `json:"egress_bytes"` // исходящий трафик
	
	Tenant   Tenant   `json:"tenant" gorm:"foreignKey:TenantID"`
	Service  Service  `json:"service" gorm:"foreignKey:ServiceID"`
	Revision *Revision `json:"revision" gorm:"foreignKey:RevisionID"`
}

// Тарифный план по модели Yandex Cloud
type PricingPlan struct {
	ID                     uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name                   string    `json:"name" gorm:"not null"`
	TenantID               *uuid.UUID `json:"tenant_id"` // NULL = общий план
	Currency               string    `json:"currency" gorm:"default:'RUB'"`
	
	// Основные цены (базируются на Yandex Cloud)
	PricePerMillionInvocations float64 `json:"price_per_million_invocations"` // 17.28 ₽
	PricePerGBHour            float64 `json:"price_per_gb_hour"`             // 5.9076 ₽
	PricePerColdStart         float64 `json:"price_per_cold_start"`          // 0 (пока не тарифицируется)
	PricePerGBEgress          float64 `json:"price_per_gb_egress"`           // 1.6524 ₽
	
	// Дополнительные опции
	PricePerGBHourProvisioned float64 `json:"price_per_gb_hour_provisioned"` // 1.296 ₽ (время простоя)
	PricePerGBHourActive      float64 `json:"price_per_gb_hour_active"`      // 2.484 ₽ (время выполнения)
	
	// Free tier (обнуляется каждый месяц)
	FreeTierInvocations    int64   `json:"free_tier_invocations"`     // 1,000,000
	FreeTierGBHours        float64 `json:"free_tier_gb_hours"`        // 10.0
	FreeTierEgressGB       float64 `json:"free_tier_egress_gb"`       // 100.0
	
	Active     bool      `json:"active" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at"`
	
	Tenant *Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
}

type Bill struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"not null"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	TotalAmount float64   `json:"total_amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status" gorm:"default:'draft'"` // draft, final, paid
	LineItems   JSONB     `json:"line_items" gorm:"type:jsonb"`
	CreatedAt   time.Time `json:"created_at"`
	
	Tenant Tenant `json:"tenant" gorm:"foreignKey:TenantID"`
}

type BillingLineItem struct {
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	FreeTierUsed   float64 `json:"free_tier_used"`
	BillableAmount float64 `json:"billable_amount"`
	TotalCost      float64 `json:"total_cost"`
	Currency       string  `json:"currency"`
}

type BillingResult struct {
	TenantID        uuid.UUID           `json:"tenant_id"`
	PeriodStart     time.Time           `json:"period_start"`
	PeriodEnd       time.Time           `json:"period_end"`
	LineItems       []BillingLineItem   `json:"line_items"`
	TotalCost       float64             `json:"total_cost"`
	Currency        string              `json:"currency"`
	FreeTierSummary FreeTierSummary     `json:"free_tier_summary"`
}

type FreeTierSummary struct {
	InvocationsUsed  int64   `json:"invocations_used"`
	InvocationsLimit int64   `json:"invocations_limit"`
	GBHoursUsed      float64 `json:"gb_hours_used"`
	GBHoursLimit     float64 `json:"gb_hours_limit"`
	EgressGBUsed     float64 `json:"egress_gb_used"`
	EgressGBLimit    float64 `json:"egress_gb_limit"`
}
