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
	billingService services.BillingService
	metricsService services.MetricsService
}

func NewHandler() Handler {
	return Handler{
		billingService: services.NewBillingService(),
		metricsService: services.NewMetricsService(),
	}
}

// Tenants
func (h Handler) CreateTenant(c *gin.Context) {
	var t models.Tenant
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if err := database.DB.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h Handler) GetTenants(c *gin.Context) {
	var tenants []models.Tenant
	if err := database.DB.Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tenants)
}

func (h Handler) GetTenant(c *gin.Context) {
	id := c.Param("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var t models.Tenant
	if err := database.DB.First(&t, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// Services
func (h Handler) CreateService(c *gin.Context) {
	var s models.Service
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if err := database.DB.Create(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s)
}

func (h Handler) GetServices(c *gin.Context) {
	var list []models.Service
	q := database.DB
	if v := c.Query("tenant_id"); v != "" {
		q = q.Where("tenant_id = ?", v)
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Usage aggregates
func (h Handler) GetUsageAggregates(c *gin.Context) {
	var aggs []models.UsageAggregate
	q := database.DB

	if s := c.Query("start_time"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			q = q.Where("window_start >= ?", t)
		}
	}
	if s := c.Query("end_time"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			q = q.Where("window_end <= ?", t)
		}
	}
	if v := c.Query("tenant_id"); v != "" {
		q = q.Where("tenant_id = ?", v)
	}
	if v := c.Query("service_id"); v != "" {
		q = q.Where("service_id = ?", v)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	if err := q.Offset(offset).Limit(limit).Order("window_start DESC").Find(&aggs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": aggs, "page": page, "limit": limit})
}

// Metrics ingest
func (h Handler) IngestMetrics(c *gin.Context) {
	var batch []models.UsageRaw
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.metricsService.IngestMetrics(batch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok", "count": len(batch)})
}

// Billing
func (h Handler) CalculateCost(c *gin.Context) {
	var req struct {
		TenantID  uuid.UUID `json:"tenant_id" binding:"required"`
		StartTime time.Time `json:"start_time" binding:"required"`
		EndTime   time.Time `json:"end_time" binding:"required"`
		ServiceID uuid.UUID `json:"service_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.billingService.CalculateCost(req.TenantID, req.StartTime, req.EndTime, req.ServiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h Handler) GenerateBill(c *gin.Context) {
	var req struct {
		TenantID  uuid.UUID `json:"tenant_id" binding:"required"`
		StartTime time.Time `json:"start_time" binding:"required"`
		EndTime   time.Time `json:"end_time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Для простоты здесь можно вызывать CalculateCost и потом сохранять Bill,
	// но в минимальном примере опущено.
	c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
