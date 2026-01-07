package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

func (h Handler) GetPricingPlans(c *gin.Context) {
	var plans []models.PricingPlan
	q := database.DB

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

	var plan models.PricingPlan
	if err := database.DB.First(&plan, "id = ?", planID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing plan not found"})
		return
	}

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
