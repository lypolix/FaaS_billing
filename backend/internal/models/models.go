package models

import (
    "time"
    "encoding/json"
    "github.com/google/uuid"
    "database/sql/driver"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
    return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
    bytes, ok := value.([]byte)
    if !ok {
        return nil
    }
    return json.Unmarshal(bytes, j)
}

// Tenant - арендатор
type Tenant struct {
    ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name         string    `json:"name" gorm:"not null"`
    Currency     string    `json:"currency" gorm:"default:'RUB'"`
    TaxRate      float64   `json:"tax_rate" gorm:"default:0.20"`
    Labels       JSONB     `json:"labels" gorm:"type:jsonb"`
    Status       string    `json:"status" gorm:"default:'active'"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// Service - Knative сервис
type Service struct {
    ID                  uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID           uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null"`
    Name               string    `json:"name" gorm:"not null"`
    Namespace          string    `json:"namespace" gorm:"not null"`
    Labels             JSONB     `json:"labels" gorm:"type:jsonb"`
    AutoscalingMetric  string    `json:"autoscaling_metric" gorm:"default:'concurrency'"`
    AutoscalingTarget  int       `json:"autoscaling_target" gorm:"default:100"`
    MinScale           int       `json:"min_scale" gorm:"default:0"`
    MaxScale           int       `json:"max_scale" gorm:"default:1000"`
    ProvisionedConcurrency *int  `json:"provisioned_concurrency"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
    
    Tenant             Tenant     `json:"tenant" gorm:"foreignKey:TenantID"`
    Revisions          []Revision `json:"revisions" gorm:"foreignKey:ServiceID"`
}

// Revision - версия сервиса
type Revision struct {
    ID                   uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    ServiceID           uuid.UUID `json:"service_id" gorm:"type:uuid;not null"`
    RevisionName        string    `json:"revision_name" gorm:"not null"`
    Image               string    `json:"image"`
    ResourcesMemoryMB   int       `json:"resources_memory_mb" gorm:"default:128"`
    ResourcesCPUCores   float64   `json:"resources_cpu_cores" gorm:"default:0.1"`
    ScaleToZero         bool      `json:"scale_to_zero" gorm:"default:true"`
    Labels              JSONB     `json:"labels" gorm:"type:jsonb"`
    CreatedAt           time.Time `json:"created_at"`
    RetiredAt           *time.Time `json:"retired_at"`
    
    Service             Service   `json:"service" gorm:"foreignKey:ServiceID"`
}

// PricingPlan - тарифный план
type PricingPlan struct {
    ID                         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name                       string     `json:"name" gorm:"not null"`
    Description                string     `json:"description"`
    Currency                   string     `json:"currency" gorm:"default:'RUB'"`
    EffectiveFrom             time.Time  `json:"effective_from"`
    EffectiveThrough          *time.Time `json:"effective_through"`
    PricePerInvocation        float64    `json:"price_per_invocation" gorm:"default:0.0"`
    PricePerMBMs              float64    `json:"price_per_mb_ms" gorm:"default:0.0"`
    PricePerExecMs            float64    `json:"price_per_exec_ms" gorm:"default:0.0"`
    PricePerColdStart         float64    `json:"price_per_cold_start" gorm:"default:0.0"`
    PricePerCPUCoreSec        float64    `json:"price_per_cpu_core_sec" gorm:"default:0.0"`
    PricePerGBEgress          float64    `json:"price_per_gb_egress" gorm:"default:0.0"`
    ProvisionedConcurrencyHourPrice float64 `json:"provisioned_concurrency_hour_price" gorm:"default:0.0"`
    FreeTierInvocations       int64      `json:"free_tier_invocations" gorm:"default:1000000"`
    FreeTierMBMs              int64      `json:"free_tier_mb_ms" gorm:"default:400000"`
    CreatedAt                 time.Time  `json:"created_at"`
    UpdatedAt                 time.Time  `json:"updated_at"`
}

// UsageRaw - сырые метрики
type UsageRaw struct {
    ID           int64     `json:"id" gorm:"primary_key;auto_increment"`
    Timestamp    time.Time `json:"timestamp" gorm:"not null;index:idx_usage_raw_ts"`
    Source       string    `json:"source" gorm:"not null"`
    Metric       string    `json:"metric" gorm:"not null;index:idx_usage_raw_metric"`
    Value        float64   `json:"value" gorm:"not null"`
    Unit         string    `json:"unit" gorm:"not null"`
    Labels       JSONB     `json:"labels" gorm:"type:jsonb;not null"`
    Temporality  string    `json:"temporality" gorm:"default:'delta'"`
    IngestID     uuid.UUID `json:"ingest_id" gorm:"type:uuid"`
}

// UsageAggregate - агрегированные данные
type UsageAggregate struct {
    ID              uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    WindowStart     time.Time `json:"window_start" gorm:"not null;index:idx_usage_agg_window"`
    WindowEnd       time.Time `json:"window_end" gorm:"not null"`
    TenantID        uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index:idx_usage_agg_tenant"`
    ServiceID       uuid.UUID `json:"service_id" gorm:"type:uuid;not null;index:idx_usage_agg_service"`  
    RevisionID      uuid.UUID `json:"revision_id" gorm:"type:uuid;not null"`
    Invocations     int64     `json:"invocations" gorm:"default:0"`
    DurationMsSum   int64     `json:"duration_ms_sum" gorm:"default:0"`
    DurationMsP50   float64   `json:"duration_ms_p50" gorm:"default:0"`
    DurationMsP95   float64   `json:"duration_ms_p95" gorm:"default:0"`
    MemMBMsSum      int64     `json:"mem_mb_ms_sum" gorm:"default:0"`
    CPUCoreSecSum   float64   `json:"cpu_core_sec_sum" gorm:"default:0"`
    ColdStarts      int64     `json:"cold_starts" gorm:"default:0"`
    EgressGB        float64   `json:"egress_gb" gorm:"default:0"`
    Errors5xx       int64     `json:"errors_5xx" gorm:"default:0"`
    ConcurrencyAvg  float64   `json:"concurrency_avg" gorm:"default:0"`
    PCHours         float64   `json:"pc_hours" gorm:"default:0"`
    EvidenceSpan    []int64   `json:"evidence_span" gorm:"type:integer[]"`
    
    Tenant          Tenant    `json:"tenant" gorm:"foreignKey:TenantID"`
    Service         Service   `json:"service" gorm:"foreignKey:ServiceID"`
    Revision        Revision  `json:"revision" gorm:"foreignKey:RevisionID"`
}

// Bill - счёт
type Bill struct {
    ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID        uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null"`
    PeriodStart     time.Time `json:"period_start" gorm:"not null"`
    PeriodEnd       time.Time `json:"period_end" gorm:"not null"`
    Currency        string    `json:"currency" gorm:"not null"`
    SubTotal        float64   `json:"sub_total" gorm:"not null"`
    TaxAmount       float64   `json:"tax_amount" gorm:"not null"`
    Total           float64   `json:"total" gorm:"not null"`
    Status          string    `json:"status" gorm:"default:'draft'"`
    AppliedPlanSnapshot JSONB `json:"applied_plan_snapshot" gorm:"type:jsonb"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    
    Tenant          Tenant     `json:"tenant" gorm:"foreignKey:TenantID"`
    LineItems       []LineItem `json:"line_items" gorm:"foreignKey:BillID"`
}

// LineItem - позиция счёта
type LineItem struct {
    ID                  uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    BillID             uuid.UUID   `json:"bill_id" gorm:"type:uuid;not null"`
    ScopeType          string      `json:"scope_type" gorm:"not null"`
    ScopeID            uuid.UUID   `json:"scope_id" gorm:"type:uuid;not null"`
    MetricType         string      `json:"metric_type" gorm:"not null"`
    Quantity           float64     `json:"quantity" gorm:"not null"`
    Unit               string      `json:"unit" gorm:"not null"`
    UnitPrice          float64     `json:"unit_price" gorm:"not null"`
    Amount             float64     `json:"amount" gorm:"not null"`
    UsageAggregateIDs  []uuid.UUID `json:"usage_aggregate_ids" gorm:"type:uuid[]"`
    Notes              string      `json:"notes"`
    
    Bill               Bill        `json:"bill" gorm:"foreignKey:BillID"`
}
