package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
