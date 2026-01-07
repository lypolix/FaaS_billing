package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

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
