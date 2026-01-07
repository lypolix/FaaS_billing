package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	// ML proxy 
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


	func (h Handler) UploadServiceArtifact(c *gin.Context) {
		id := c.Param("id")
		uid, err := uuid.Parse(id)
		if err != nil {
		  c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		  return
		}
	  
		var s models.Service
		if err := database.DB.First(&s, "id = ?", uid).Error; err != nil {
		  c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		  return
		}
	  
		file, err := c.FormFile("file")
		if err != nil {
		  c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		  return
		}
	  
		baseDir := os.Getenv("ARTIFACTS_DIR")
		if baseDir == "" {
		  baseDir = "./data/artifacts"
		}
	  
		safeName := filepath.Base(file.Filename)
		dir := filepath.Join(baseDir, uid.String())
		if err := os.MkdirAll(dir, 0o755); err != nil {
		  c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create artifacts dir"})
		  return
		}
	  
		dst := filepath.Join(dir, safeName)
	
		if err := c.SaveUploadedFile(file, dst); err != nil {
		  c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + err.Error()})
		  return
		}
	
		f, err := os.Open(dst)
		if err != nil {
		  c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open saved file"})
		  return
		}
		defer f.Close()
	  
		hsh := sha256.New()
		if _, err := io.Copy(hsh, f); err != nil {
		  c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash file"})
		  return
		}
		sha := hex.EncodeToString(hsh.Sum(nil))
	  
		s.ArtifactPath = dst
		s.ArtifactName = safeName
		s.ArtifactSize = file.Size
		s.ArtifactSHA = sha
	  
		if err := database.DB.Save(&s).Error; err != nil {
		  c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update service"})
		  return
		}
	  
		artifactURL := "/api/v1/artifacts/" + uid.String() + "/" + safeName
		c.JSON(http.StatusOK, gin.H{
		  "service_id":   s.ID,
		  "artifact_url": artifactURL,
		  "size_bytes":   s.ArtifactSize,
		  "sha256":       s.ArtifactSHA,
		})
	  }

	  func (h Handler) DownloadArtifact(c *gin.Context) {
		serviceID := c.Param("service_id")
		uid, err := uuid.Parse(serviceID)
		if err != nil {
		  c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		  return
		}
	  
		var s models.Service
		if err := database.DB.First(&s, "id = ?", uid).Error; err != nil {
		  c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		  return
		}
		if s.ArtifactPath == "" {
		  c.JSON(http.StatusNotFound, gin.H{"error": "artifact not uploaded"})
		  return
		}
	  
		c.File(s.ArtifactPath)
	  }
	  
	  
	  func (h Handler) GetPricingPlans(c *gin.Context) {
		var plans []models.PricingPlan
		q := database.DB
	
		// опционально: только активные
		if c.Query("active") == "true" {
			q = q.Where("active = ?", true)
		}
	
		if err := q.Order("created_at DESC").Find(&plans).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, plans)
	}

	func (h Handler) SetTenantPricingPlan(c *gin.Context) {
		tenantIDStr := c.Param("id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}
	
		var req struct {
			PricingPlanID string `json:"pricing_plan_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	
		planID, err := uuid.Parse(req.PricingPlanID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing_plan_id"})
			return
		}
	
		// проверить что план существует
		var plan models.PricingPlan
		if err := database.DB.First(&plan, "id = ?", planID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pricing plan not found"})
			return
		}
	
		// обновить tenant
		if err := database.DB.Model(&models.Tenant{}).
			Where("id = ?", tenantID).
			Update("pricing_plan_id", planID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update tenant: " + err.Error()})
			return
		}
	
		c.JSON(http.StatusOK, gin.H{"message": "pricing plan updated"})
	}
	
	func (h Handler) GetTenantPricingPlan(c *gin.Context) {
		tenantIDStr := c.Param("id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
			return
		}
	
		var t models.Tenant
		if err := database.DB.First(&t, "id = ?", tenantID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		if t.PricingPlanID == nil {
			c.JSON(http.StatusOK, gin.H{"pricing_plan_id": nil})
			return
		}
	
		var plan models.PricingPlan
		if err := database.DB.First(&plan, "id = ?", *t.PricingPlanID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pricing plan not found"})
			return
		}
	
		c.JSON(http.StatusOK, plan)
	}
		
