package handlers

import (
    "net/http"
    "strconv"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    
    "github.com/lypolix/FaaS-billing/internal/database"
    "github.com/lypolix/FaaS-billing/internal/models"
    "github.com/lypolix/FaaS-billing/internal/services"
)

type Handler struct {
    billingService *services.BillingService
    metricsService *services.MetricsService
}

func NewHandler() *Handler {
    return &Handler{
        billingService: services.NewBillingService(),
        metricsService: services.NewMetricsService(),
    }
}

// Tenants
func (h *Handler) CreateTenant(c *gin.Context) {
    var tenant models.Tenant
    if err := c.ShouldBindJSON(&tenant); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    result := database.DB.Create(&tenant)
    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, tenant)
}

func (h *Handler) GetTenants(c *gin.Context) {
    var tenants []models.Tenant
    result := database.DB.Find(&tenants)
    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
        return
    }
    
    c.JSON(http.StatusOK, tenants)
}

func (h *Handler) GetTenant(c *gin.Context) {
    id := c.Param("id")
    tenantID, err := uuid.Parse(id)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
        return
    }
    
    var tenant models.Tenant
    result := database.DB.First(&tenant, tenantID)
    if result.Error != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
        return
    }
    
    c.JSON(http.StatusOK, tenant)
}

// Services
func (h *Handler) CreateService(c *gin.Context) {
    var service models.Service
    if err := c.ShouldBindJSON(&service); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    result := database.DB.Create(&service)
    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, service)
}

func (h *Handler) GetServices(c *gin.Context) {
    var services []models.Service
    query := database.DB.Preload("Tenant").Preload("Revisions")
    
    // Filter by tenant if provided
    if tenantID := c.Query("tenant_id"); tenantID != "" {
        query = query.Where("tenant_id = ?", tenantID)
    }
    
    result := query.Find(&services)
    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
        return
    }
    
    c.JSON(http.StatusOK, services)
}

// Usage aggregates
func (h *Handler) GetUsageAggregates(c *gin.Context) {
    var aggregates []models.UsageAggregate
    query := database.DB.Preload("Tenant").Preload("Service").Preload("Revision")
    
    // Time range filters
    if startTime := c.Query("start_time"); startTime != "" {
        if t, err := time.Parse(time.RFC3339, startTime); err == nil {
            query = query.Where("window_start >= ?", t)
        }
    }
    
    if endTime := c.Query("end_time"); endTime != "" {
        if t, err := time.Parse(time.RFC3339, endTime); err == nil {
            query = query.Where("window_end <= ?", t)
        }
    }
    
    // Entity filters
    if tenantID := c.Query("tenant_id"); tenantID != "" {
        query = query.Where("tenant_id = ?", tenantID)
    }
    
    if serviceID := c.Query("service_id"); serviceID != "" {
        query = query.Where("service_id = ?", serviceID)
    }
    
    // Pagination
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
    offset := (page - 1) * limit
    
    result := query.Offset(offset).Limit(limit).Order("window_start DESC").Find(&aggregates)
    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": aggregates,
        "page": page,
        "limit": limit,
    })
}

// Billing
func (h *Handler) CalculateCost(c *gin.Context) {
    var request struct {
        TenantID  uuid.UUID `json:"tenant_id" binding:"required"`
        StartTime time.Time `json:"start_time" binding:"required"`
        EndTime   time.Time `json:"end_time" binding:"required"`
        ServiceID *uuid.UUID `json:"service_id,omitempty"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    cost, err := h.billingService.CalculateCost(request.TenantID, request.StartTime, request.EndTime, request.ServiceID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, cost)
}

func (h *Handler) GenerateBill(c *gin.Context) {
    var request struct {
        TenantID  uuid.UUID `json:"tenant_id" binding:"required"`
        StartTime time.Time `json:"start_time" binding:"required"`
        EndTime   time.Time `json:"end_time" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    bill, err := h.billingService.GenerateBill(request.TenantID, request.StartTime, request.EndTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, bill)
}

// Metrics ingestion
func (h *Handler) IngestMetrics(c *gin.Context) {
    var metrics []models.UsageRaw
    if err := c.ShouldBindJSON(&metrics); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    err := h.metricsService.IngestMetrics(metrics)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "Metrics ingested successfully", "count": len(metrics)})
}
