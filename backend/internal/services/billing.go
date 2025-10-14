package services

import (
    "time"
    
    "github.com/google/uuid"
    "github.com/lypolix/FaaS-billing/internal/database"
    "github.com/lypolix/FaaS-billing/internal/models"
)

type BillingService struct{}

func NewBillingService() *BillingService {
    return &BillingService{}
}

type CostBreakdown struct {
    TenantID        uuid.UUID                    `json:"tenant_id"`
    PeriodStart     time.Time                    `json:"period_start"`
    PeriodEnd       time.Time                    `json:"period_end"`
    Services        []ServiceCost                `json:"services"`
    TotalCost       float64                      `json:"total_cost"`
    Currency        string                       `json:"currency"`
}

type ServiceCost struct {
    ServiceID       uuid.UUID     `json:"service_id"`
    ServiceName     string        `json:"service_name"`
    Components      []CostComponent `json:"components"`
    TotalCost       float64       `json:"total_cost"`
}

type CostComponent struct {
    Type        string  `json:"type"`        // invocations, mb_ms, cold_start, etc.
    Quantity    float64 `json:"quantity"`
    UnitPrice   float64 `json:"unit_price"`
    Amount      float64 `json:"amount"`
    FreeTier    float64 `json:"free_tier"`
    Unit        string  `json:"unit"`
}

func (bs *BillingService) CalculateCost(tenantID uuid.UUID, startTime, endTime time.Time, serviceID *uuid.UUID) (*CostBreakdown, error) {
    // Get aggregates for the period
    var aggregates []models.UsageAggregate
    query := database.DB.Preload("Service").Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", tenantID, startTime, endTime)
    
    if serviceID != nil {
        query = query.Where("service_id = ?", *serviceID)
    }
    
    if err := query.Find(&aggregates).Error; err != nil {
        return nil, err
    }
    
    // Get active pricing plan for tenant
    var pricingPlan models.PricingPlan
    if err := database.DB.Where("effective_from <= ? AND (effective_through IS NULL OR effective_through >= ?)", startTime, endTime).First(&pricingPlan).Error; err != nil {
        return nil, err
    }
    
    // Group by service
    serviceAggregates := make(map[uuid.UUID][]models.UsageAggregate)
    for _, agg := range aggregates {
        serviceAggregates[agg.ServiceID] = append(serviceAggregates[agg.ServiceID], agg)
    }
    
    breakdown := &CostBreakdown{
        TenantID:    tenantID,
        PeriodStart: startTime,
        PeriodEnd:   endTime,
        Currency:    pricingPlan.Currency,
        Services:    make([]ServiceCost, 0),
    }
    
    totalCost := 0.0
    
    for serviceID, serviceAggregate := range serviceAggregates {
        serviceCost := bs.calculateServiceCost(serviceAggregate, &pricingPlan)
        serviceCost.ServiceID = serviceID
        if len(serviceAggregate) > 0 {
            serviceCost.ServiceName = serviceAggregate[0].Service.Name
        }
        
        breakdown.Services = append(breakdown.Services, serviceCost)
        totalCost += serviceCost.TotalCost
    }
    
    breakdown.TotalCost = totalCost
    
    return breakdown, nil
}

func (bs *BillingService) calculateServiceCost(aggregates []models.UsageAggregate, plan *models.PricingPlan) ServiceCost {
    // Sum all metrics across windows
    totalInvocations := int64(0)
    totalMBMs := int64(0)
    totalColdStarts := int64(0)
    totalCPUCoreSec := 0.0
    totalEgressGB := 0.0
    totalPCHours := 0.0
    totalDurationMs := int64(0)
    
    for _, agg := range aggregates {
        totalInvocations += agg.Invocations
        totalMBMs += agg.MemMBMsSum
        totalColdStarts += agg.ColdStarts
        totalCPUCoreSec += agg.CPUCoreSecSum
        totalEgressGB += agg.EgressGB
        totalPCHours += agg.PCHours
        totalDurationMs += agg.DurationMsSum
    }
    
    components := []CostComponent{}
    totalCost := 0.0
    
    // Invocations cost
    if plan.PricePerInvocation > 0 {
        billableInvocations := float64(totalInvocations)
        freeTier := float64(plan.FreeTierInvocations)
        if billableInvocations > freeTier {
            billableInvocations -= freeTier
        } else {
            billableInvocations = 0
        }
        
        amount := billableInvocations * plan.PricePerInvocation
        totalCost += amount
        
        components = append(components, CostComponent{
            Type:      "invocations",
            Quantity:  float64(totalInvocations),
            UnitPrice: plan.PricePerInvocation,
            Amount:    amount,
            FreeTier:  freeTier,
            Unit:      "requests",
        })
    }
    
    // Memory cost (MB-ms)
    if plan.PricePerMBMs > 0 {
        billableMBMs := float64(totalMBMs)
        freeTier := float64(plan.FreeTierMBMs)
        if billableMBMs > freeTier {
            billableMBMs -= freeTier
        } else {
            billableMBMs = 0
        }
        
        amount := billableMBMs * plan.PricePerMBMs
        totalCost += amount
        
        components = append(components, CostComponent{
            Type:      "mb_ms",
            Quantity:  float64(totalMBMs),
            UnitPrice: plan.PricePerMBMs,
            Amount:    amount,
            FreeTier:  freeTier,
            Unit:      "MB-ms",
        })
    }
    
    // Execution time cost
    if plan.PricePerExecMs > 0 {
        amount := float64(totalDurationMs) * plan.PricePerExecMs
        totalCost += amount
        
        components = append(components, CostComponent{
            Type:      "exec_ms",
            Quantity:  float64(totalDurationMs),
            UnitPrice: plan.PricePerExecMs,
            Amount:    amount,
            Unit:      "ms",
        })
    }
    
    // Cold start cost
    if plan.PricePerColdStart > 0 {
        amount := float64(totalColdStarts) * plan.PricePerColdStart
        totalCost += amount
        
        components = append(components, CostComponent{
            Type:      "cold_start",
            Quantity:  float64(totalColdStarts),
            UnitPrice: plan.PricePerColdStart,
            Amount:    amount,
            Unit:      "starts",
        })
    }
    
    return ServiceCost{
        Components: components,
        TotalCost:  totalCost,
    }
}

func (bs *BillingService) GenerateBill(tenantID uuid.UUID, startTime, endTime time.Time) (*models.Bill, error) {
    // Calculate cost breakdown
    breakdown, err := bs.CalculateCost(tenantID, startTime, endTime, nil)
    if err != nil {
        return nil, err
    }
    
    // Get tenant for tax rate
    var tenant models.Tenant
    if err := database.DB.First(&tenant, tenantID).Error; err != nil {
        return nil, err
    }
    
    // Create bill
    subTotal := breakdown.TotalCost
    taxAmount := subTotal * tenant.TaxRate
    total := subTotal + taxAmount
    
    bill := &models.Bill{
        TenantID:    tenantID,
        PeriodStart: startTime,
        PeriodEnd:   endTime,
        Currency:    breakdown.Currency,
        SubTotal:    subTotal,
        TaxAmount:   taxAmount,
        Total:       total,
        Status:      "issued",
    }
    
    if err := database.DB.Create(bill).Error; err != nil {
        return nil, err
    }
    
    // Create line items
    for _, service := range breakdown.Services {
        for _, component := range service.Components {
            lineItem := models.LineItem{
                BillID:     bill.ID,
                ScopeType:  "service",
                ScopeID:    service.ServiceID,
                MetricType: component.Type,
                Quantity:   component.Quantity,
                Unit:       component.Unit,
                UnitPrice:  component.UnitPrice,
                Amount:     component.Amount,
            }
            
            if err := database.DB.Create(&lineItem).Error; err != nil {
                return nil, err
            }
        }
    }
    
    // Load with associations
    if err := database.DB.Preload("LineItems").Preload("Tenant").First(bill, bill.ID).Error; err != nil {
        return nil, err
    }
    
    return bill, nil
}
