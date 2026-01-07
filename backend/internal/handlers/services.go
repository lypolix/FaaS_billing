package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

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
