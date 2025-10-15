package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

type BillingService struct{}

func NewBillingService() BillingService { return BillingService{} }

type CostComponent struct {
	Type      string  `json:"type"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Amount    float64 `json:"amount"`
	FreeTier  float64 `json:"freetier,omitempty"`
	Unit      string  `json:"unit"`
}

type ServiceCost struct {
	ServiceID   uuid.UUID       `json:"serviceid"`
	ServiceName string          `json:"servicename"`
	Components  []CostComponent `json:"components"`
	TotalCost   float64         `json:"totalcost"`
}

type CostBreakdown struct {
	TenantID    uuid.UUID    `json:"tenantid"`
	PeriodStart time.Time    `json:"periodstart"`
	PeriodEnd   time.Time    `json:"periodend"`
	Services    []ServiceCost`json:"services"`
	TotalCost   float64      `json:"totalcost"`
	Currency    string       `json:"currency"`
}

func (bs BillingService) CalculateCost(tenantID uuid.UUID, startTime, endTime time.Time, serviceID uuid.UUID) (CostBreakdown, error) {
	var aggregates []models.UsageAggregate
	q := database.DB.
		Where("tenant_id = ? AND window_start >= ? AND window_end <= ?", tenantID, startTime, endTime)
	if serviceID != uuid.Nil {
		q = q.Where("service_id = ?", serviceID)
	}
	if err := q.Find(&aggregates).Error; err != nil {
		return CostBreakdown{}, err
	}

	var plan models.PricingPlan
	if err := database.DB.
		Where("tenant_id = ? AND effective_from <= ? AND (effective_through IS NULL OR effective_through >= ?)",
			tenantID, startTime, endTime).
		Order("effective_from DESC").
		First(&plan).Error; err != nil {
		plan = models.PricingPlan{
			TenantID:            tenantID,
			Currency:            "RUB",
			PricePerInvocation:  0.0000035,
			PricePerMBMs:        0.000016,
			PricePerExecMs:      0,
			PricePerColdStart:   0.001,
			FreeTierInvocations: 1000000,
			FreeTierMBMs:        400000,
		}
	}

	group := map[uuid.UUID][]models.UsageAggregate{}
	for _, a := range aggregates {
		group[a.ServiceID] = append(group[a.ServiceID], a)
	}

	out := CostBreakdown{
		TenantID:    tenantID,
		PeriodStart: startTime,
		PeriodEnd:   endTime,
		Currency:    plan.Currency,
	}
	var total float64

	// имена сервисов
	var services []models.Service
	_ = database.DB.Where("tenant_id = ?", tenantID).Find(&services).Error
	nameByID := map[uuid.UUID]string{}
	for _, s := range services {
		nameByID[s.ID] = s.Name
	}

	for id, arr := range group {
		sc := bs.calculateServiceCost(arr, plan)
		sc.ServiceID = id
		sc.ServiceName = nameByID[id]
		out.Services = append(out.Services, sc)
		total += sc.TotalCost
	}
	out.TotalCost = total
	return out, nil
}

func (bs BillingService) calculateServiceCost(aggs []models.UsageAggregate, plan models.PricingPlan) ServiceCost {
	var (
		totalInvocations int64
		totalMBMs        int64
		totalColdStarts  int64
		totalExecMs      int64
	)
	for _, a := range aggs {
		totalInvocations += a.Invocations
		totalMBMs += a.MemMBMsSum
		totalColdStarts += a.ColdStarts
		totalExecMs += a.DurationMsSum
	}

	var comps []CostComponent
	var total float64

	// Invocations
	if plan.PricePerInvocation > 0 {
		billable := float64(totalInvocations)
		free := plan.FreeTierInvocations
		if billable > free {
			billable -= free
		} else {
			billable = 0
		}
		amount := billable * plan.PricePerInvocation
		total += amount
		comps = append(comps, CostComponent{
			Type:      "invocations",
			Quantity:  float64(totalInvocations),
			UnitPrice: plan.PricePerInvocation,
			Amount:    amount,
			FreeTier:  free,
			Unit:      "requests",
		})
	}

	// Memory MB-ms
	if plan.PricePerMBMs > 0 {
		billable := float64(totalMBMs)
		free := plan.FreeTierMBMs
		if billable > free {
			billable -= free
		} else {
			billable = 0
		}
		amount := billable * plan.PricePerMBMs
		total += amount
		comps = append(comps, CostComponent{
			Type:      "mbms",
			Quantity:  float64(totalMBMs),
			UnitPrice: plan.PricePerMBMs,
			Amount:    amount,
			FreeTier:  free,
			Unit:      "MB-ms",
		})
	}

	// Exec time ms
	if plan.PricePerExecMs > 0 {
		amount := float64(totalExecMs) * plan.PricePerExecMs
		total += amount
		comps = append(comps, CostComponent{
			Type:      "execms",
			Quantity:  float64(totalExecMs),
			UnitPrice: plan.PricePerExecMs,
			Amount:    amount,
			Unit:      "ms",
		})
	}

	// Cold starts
	if plan.PricePerColdStart > 0 {
		amount := float64(totalColdStarts) * plan.PricePerColdStart
		total += amount
		comps = append(comps, CostComponent{
			Type:      "coldstart",
			Quantity:  float64(totalColdStarts),
			UnitPrice: plan.PricePerColdStart,
			Amount:    amount,
			Unit:      "starts",
		})
	}

	return ServiceCost{
		Components: comps,
		TotalCost:  total,
	}
}
