	package handlers

	import (
		"io"
		"net/http"
		"time"

		"github.com/gin-gonic/gin"
		"github.com/google/uuid"

		"github.com/lypolix/FaaS-billing/internal/database"
		"github.com/lypolix/FaaS-billing/internal/models"
		"github.com/lypolix/FaaS-billing/internal/services"
	)

	type Handler struct {
		BillingService *services.BillingService
		MetricsService *services.MetricsService
	}

	func NewHandler() Handler {
		return Handler{
			BillingService: services.NewBillingService(database.DB),
			MetricsService: services.NewMetricsService(database.DB),
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

		if err := q.Order("window_start DESC").Find(&aggs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": aggs})
	}

	// Metrics ingest
	func (h Handler) IngestMetrics(c *gin.Context) {
		var batch []models.UsageRaw
		if err := c.ShouldBindJSON(&batch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.MetricsService.IngestMetrics(batch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok", "count": len(batch)})
	}

	// Metrics aggregate (ручной запуск)
	func (h Handler) AggregateMetrics(c *gin.Context) {
		var req struct {
			StartTime  time.Time `json:"start_time" binding:"required"`
			EndTime    time.Time `json:"end_time" binding:"required"`
			WindowSize string    `json:"window_size" binding:"required"` // "5m","1h","1d"
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.MetricsService.AggregateMetrics(req.StartTime, req.EndTime, req.WindowSize); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "aggregation completed"})
	}

	// Billing
	func (h Handler) CalculateCost(c *gin.Context) {
		var req struct {
			TenantID  string    `json:"tenant_id" binding:"required"`
			StartTime time.Time `json:"start_time" binding:"required"`
			EndTime   time.Time `json:"end_time" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		out, err := h.BillingService.CalculateBill(req.TenantID, req.StartTime, req.EndTime)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}

	func (h Handler) GenerateBill(c *gin.Context) {
		var req struct {
			TenantID  string    `json:"tenant_id" binding:"required"`
			StartTime time.Time `json:"start_time" binding:"required"`
			EndTime   time.Time `json:"end_time" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		result, err := h.BillingService.CalculateBill(req.TenantID, req.StartTime, req.EndTime)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if _, err := h.BillingService.SaveBill(result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save bill: " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "bill generated", "bill": result})
	}

	// ML proxy (простой вариант)
	func (h Handler) ProxyForecast(c *gin.Context) {
		resp, err := http.Post("http://ai-forecast:8082/forecast/cost", "application/json", c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "ml service unreachable"})
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
	}
