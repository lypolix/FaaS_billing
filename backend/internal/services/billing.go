package services

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/lypolix/FaaS-billing/internal/models"
	"gorm.io/gorm"
)

type BillingService struct {
	db *gorm.DB
}

func NewBillingService(db *gorm.DB) *BillingService {
	return &BillingService{db: db}
}

// CalculateBill - основная функция расчёта стоимости по формулам Yandex Cloud
func (s *BillingService) CalculateBill(tenantID string, startTime, endTime time.Time) (*models.BillingResult, error) {
    // 1) Загружаем tenant и берём выбранный plan_id
    var tenant models.Tenant
    if err := s.db.First(&tenant, "id = ?", tenantID).Error; err != nil {
        return nil, fmt.Errorf("tenant not found: %w", err)
    }

    if tenant.PricingPlanID == nil {
        return nil, fmt.Errorf("tenant has no pricing plan assigned")
    }

    // 2) Загружаем тарифный план строго по PricingPlanID
    var pricingPlan models.PricingPlan
    if err := s.db.First(&pricingPlan, "id = ? AND active = ?", *tenant.PricingPlanID, true).Error; err != nil {
        return nil, fmt.Errorf("pricing plan not found or inactive: %w", err)
    }

    // 3) Получаем агрегированные данные за период
    var aggregates []models.UsageAggregate
    err := s.db.Where(
        "tenant_id = ? AND window_start >= ? AND window_end <= ?",
        tenantID, startTime, endTime,
    ).Find(&aggregates).Error
    if err != nil {
        return nil, fmt.Errorf("failed to get usage aggregates: %w", err)
    }

    // 4) Считаем общие показатели
    totals := s.calculateTotals(aggregates)

    // 5) Применяем биллинговые формулы
    result := &models.BillingResult{
        TenantID:    uuid.MustParse(tenantID),
        PeriodStart: startTime,
        PeriodEnd:   endTime,
        Currency:    pricingPlan.Currency,
        LineItems:   []models.BillingLineItem{},
    }

    invocationsItem := s.calculateInvocationsCost(totals.TotalInvocations, pricingPlan)
    result.LineItems = append(result.LineItems, invocationsItem)

    computeItem := s.calculateComputeCost(totals.TotalGBHours, pricingPlan)
    result.LineItems = append(result.LineItems, computeItem)

    if totals.TotalEgressGB > 0 {
        egressItem := s.calculateEgressCost(totals.TotalEgressGB, pricingPlan)
        result.LineItems = append(result.LineItems, egressItem)
    }

    coldStartsItem := models.BillingLineItem{
        Description:    "Холодные старты",
        Quantity:       float64(totals.TotalColdStarts),
        UnitPrice:      pricingPlan.PricePerColdStart,
        FreeTierUsed:   float64(totals.TotalColdStarts), 
        BillableAmount: 0,
        TotalCost:      0,
        Currency:       pricingPlan.Currency,
    }
    result.LineItems = append(result.LineItems, coldStartsItem)

    var totalCost float64
    for _, item := range result.LineItems {
        totalCost += item.TotalCost
    }
    result.TotalCost = math.Round(totalCost*100) / 100 
    

    result.FreeTierSummary = models.FreeTierSummary{
        InvocationsUsed:  totals.TotalInvocations,
        InvocationsLimit: pricingPlan.FreeTierInvocations,
        GBHoursUsed:      totals.TotalGBHours,
        GBHoursLimit:     pricingPlan.FreeTierGBHours,
        EgressGBUsed:     totals.TotalEgressGB,
        EgressGBLimit:    pricingPlan.FreeTierEgressGB,
    }

    return result, nil
}

type UsageTotals struct {
	TotalInvocations int64
	TotalGBHours     float64
	TotalEgressGB    float64
	TotalColdStarts  int64
}

func (s *BillingService) calculateTotals(aggregates []models.UsageAggregate) UsageTotals {
	totals := UsageTotals{}
	
	for _, agg := range aggregates {
		totals.TotalInvocations += agg.Invocations
		totals.TotalColdStarts += int64(agg.ColdStarts)
		
		// Переводим МБ×час в ГБ×час
		gbHours := agg.TotalMemoryMBHours / 1024.0
		totals.TotalGBHours += gbHours
		
		// Переводим bytes в GB
		egressGB := float64(agg.EgressBytes) / (1024.0 * 1024.0 * 1024.0)
		totals.TotalEgressGB += egressGB
	}
	
	return totals
}

// Формула Yandex: 17,28 ₽ × ((количество_вызовов - 1_000_000) / 1_000_000)
func (s *BillingService) calculateInvocationsCost(totalInvocations int64, plan models.PricingPlan) models.BillingLineItem {
	freeTierUsed := int64(math.Min(float64(totalInvocations), float64(plan.FreeTierInvocations)))
	billableInvocations := int64(math.Max(0, float64(totalInvocations-plan.FreeTierInvocations)))
	
	// Цена за миллион
	billableMillions := float64(billableInvocations) / 1_000_000.0
	cost := billableMillions * plan.PricePerMillionInvocations
	
	return models.BillingLineItem{
		Description:    "Вызовы функций",
		Quantity:       float64(totalInvocations),
		UnitPrice:      plan.PricePerMillionInvocations, // за миллион
		FreeTierUsed:   float64(freeTierUsed),
		BillableAmount: float64(billableInvocations),
		TotalCost:      math.Round(cost*100) / 100,
		Currency:       plan.Currency,
	}
}

// Формула Yandex: 5,9076 ₽ × (ГБ×час - 10)
func (s *BillingService) calculateComputeCost(totalGBHours float64, plan models.PricingPlan) models.BillingLineItem {
	freeTierUsed := math.Min(totalGBHours, plan.FreeTierGBHours)
	billableGBHours := math.Max(0, totalGBHours-plan.FreeTierGBHours)
	
	cost := billableGBHours * plan.PricePerGBHour
	
	return models.BillingLineItem{
		Description:    "Время выполнения функций (ГБ×час)",
		Quantity:       totalGBHours,
		UnitPrice:      plan.PricePerGBHour,
		FreeTierUsed:   freeTierUsed,
		BillableAmount: billableGBHours,
		TotalCost:      math.Round(cost*100) / 100,
		Currency:       plan.Currency,
	}
}

// Формула Yandex: 1,6524 ₽ × (ГБ_трафика - 100)
func (s *BillingService) calculateEgressCost(totalEgressGB float64, plan models.PricingPlan) models.BillingLineItem {
	freeTierUsed := math.Min(totalEgressGB, plan.FreeTierEgressGB)
	billableEgressGB := math.Max(0, totalEgressGB-plan.FreeTierEgressGB)
	
	cost := billableEgressGB * plan.PricePerGBEgress
	
	return models.BillingLineItem{
		Description:    "Исходящий трафик",
		Quantity:       totalEgressGB,
		UnitPrice:      plan.PricePerGBEgress,
		FreeTierUsed:   freeTierUsed,
		BillableAmount: billableEgressGB,
		TotalCost:      math.Round(cost*100) / 100,
		Currency:       plan.Currency,
	}
}

// Сохранить счёт в БД
func (s *BillingService) SaveBill(result *models.BillingResult) (*models.Bill, error) {
	lineItemsJSON := make(models.JSONB)
	lineItemsJSON["items"] = result.LineItems
	lineItemsJSON["free_tier"] = result.FreeTierSummary
	
	bill := &models.Bill{
		TenantID:    result.TenantID,
		PeriodStart: result.PeriodStart,
		PeriodEnd:   result.PeriodEnd,
		TotalAmount: result.TotalCost,
		Currency:    result.Currency,
		Status:      "draft",
		LineItems:   lineItemsJSON,
		CreatedAt:   time.Now(),
	}
	
	err := s.db.Create(bill).Error
	return bill, err
}

// Демонстрация расчёта как в документации Yandex
func (s *BillingService) ExampleCalculation() *models.BillingResult {
	// Пример из документации:
	// Память: 512 МБ, Вызовы: 10,000,000, Время: 800 мс каждый
	// Результат должен быть: 6,660.44 ₽
	
	memoryMB := 512.0
	invocations := int64(10_000_000)
	durationMS := 800.0
	
	// Переводим в ГБ×час
	gbHours := (memoryMB / 1024.0) * (durationMS / 3_600_000.0) * float64(invocations)
	
	plan := models.PricingPlan{
		PricePerMillionInvocations: 17.28,
		PricePerGBHour:            5.9076,
		FreeTierInvocations:       1_000_000,
		FreeTierGBHours:           10.0,
		Currency:                  "RUB",
	}
	
	result := &models.BillingResult{
		Currency: "RUB",
		LineItems: []models.BillingLineItem{
			s.calculateInvocationsCost(invocations, plan),
			s.calculateComputeCost(gbHours, plan),
		},
	}
	
	result.TotalCost = result.LineItems[0].TotalCost + result.LineItems[1].TotalCost
	
	return result
}
