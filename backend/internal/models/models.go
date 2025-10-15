package models

import (
	"time"

	"github.com/google/uuid"
)

// Tenant — включает Currency и Status для сидера
type Tenant struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	TaxRate   float64   `json:"tax_rate"`
	Status    string    `json:"status"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Service struct {
	ID                uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID          uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	AutoscalingMetric string    `json:"autoscaling_metric"`
	AutoscalingTarget int       `json:"autoscaling_target"`
	MinScale          int       `json:"min_scale"`
	MaxScale          int       `json:"max_scale"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Revision struct {
	ID                uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ServiceID         uuid.UUID `gorm:"type:uuid;index"`
	RevisionName      string
	Image             string
	ResourcesMemoryMB int
	ResourcesCPUCores float64
	ScaleToZero       bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type PricingPlan struct {
	ID                  uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TenantID            uuid.UUID `gorm:"type:uuid;index"`
	Name                string
	Description         string
	Currency            string
	PricePerInvocation  float64
	PricePerMBMs        float64
	PricePerExecMs      float64
	PricePerColdStart   float64
	FreeTierInvocations float64
	FreeTierMBMs        float64
	EffectiveFrom       time.Time
	EffectiveThrough    *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UsageRaw struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID   uuid.UUID  `gorm:"type:uuid;index" json:"tenant_id"`
	ServiceID  uuid.UUID  `gorm:"type:uuid;index" json:"service_id"`
	RevisionID *uuid.UUID `gorm:"type:uuid;index" json:"revision_id,omitempty"`

	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Temporality string    `json:"temporality"`

	Timestamp time.Time `json:"timestamp"`
	Labels    string    `json:"labels"`
	CreatedAt time.Time
}

type UsageAggregate struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TenantID   uuid.UUID  `gorm:"type:uuid;index"`
	ServiceID  uuid.UUID  `gorm:"type:uuid;index"`
	RevisionID *uuid.UUID `gorm:"type:uuid;index"`

	WindowStart time.Time `gorm:"index"`
	WindowEnd   time.Time `gorm:"index"`

	Invocations   int64
	MemMBMsSum    int64
	DurationMsSum int64
	ColdStarts    int64
	CPUCoreSecSum float64
	EgressGB      float64
	PCHours       float64

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Bill struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID    uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Currency    string    `json:"currency"`
	SubTotal    float64   `json:"sub_total"`
	TaxAmount   float64   `json:"tax_amount"`
	Total       float64   `json:"total"`
	Status      string    `json:"status"`
	LineItems   []LineItem
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type LineItem struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BillID     uuid.UUID `gorm:"type:uuid;index" json:"bill_id"`
	ScopeType  string    `json:"scope_type"`
	ScopeID    uuid.UUID `json:"scope_id"`
	MetricType string    `json:"metric_type"`
	Quantity   float64   `json:"quantity"`
	Unit       string    `json:"unit"`
	UnitPrice  float64   `json:"unit_price"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time
}
