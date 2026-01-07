package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lypolix/FaaS-billing/internal/models"
)

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
