package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
