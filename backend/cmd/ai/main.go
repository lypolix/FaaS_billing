package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/lypolix/FaaS-billing/internal/services/forecast"
)

func main() {
	r := gin.Default()

	r.POST("/forecast/cost", func(c *gin.Context) {
		var req forecast.ForecastRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		resp, err := forecast.ForecastCost(req)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, resp)
	})

	log.Println("Starting ML-forecast server on :8082")
	r.Run(":8082")
}
