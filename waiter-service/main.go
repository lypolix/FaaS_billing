package main

import (
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Метрики для биллинга
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waiter_requests_total",
			Help: "Total number of requests processed",
		},
		[]string{"method", "endpoint", "status", "tenant_id", "service_name", "revision"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "waiter_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"method", "endpoint", "tenant_id", "service_name", "revision"},
	)

	memoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "waiter_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
		[]string{"tenant_id", "service_name", "revision"},
	)

	coldStarts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waiter_cold_starts_total",
			Help: "Total number of cold starts",
		},
		[]string{"tenant_id", "service_name", "revision"},
	)
)

var (
	// Метаданные из среды Knative
	serviceName = getEnv("K_SERVICE", "waiter")
	revision    = getEnv("K_REVISION", "waiter-00001")
	tenantID    = getEnv("TENANT_ID", "demo-tenant")
	startTime   = time.Now()
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(memoryUsage)
	prometheus.MustRegister(coldStarts)

	// Фиксируем cold start при запуске pod
	coldStarts.WithLabelValues(tenantID, serviceName, revision).Inc()
}

type WaiterRequest struct {
	SleepMs   int `json:"sleep_ms" form:"sleep_ms" query:"sleep_ms"`
	MemMB     int `json:"mem_mb" form:"mem_mb" query:"mem_mb"`
	CpuSpinMs int `json:"cpu_spin_ms" form:"cpu_spin_ms" query:"cpu_spin_ms"`
}

type WaiterResponse struct {
	Request       WaiterRequest `json:"request"`
	DurationMs    int64         `json:"duration_ms"`
	MemoryAllocMB float64       `json:"memory_alloc_mb"`
	ColdStart     bool          `json:"cold_start"`
	Timestamp     int64         `json:"timestamp"`
	Metadata      Metadata      `json:"metadata"`
}

type Metadata struct {
	ServiceName string `json:"service_name"`
	Revision    string `json:"revision"`
	TenantID    string `json:"tenant_id"`
	PodName     string `json:"pod_name"`
	Uptime      string `json:"uptime"`
}

func main() {
	r := gin.Default()

	// Логи HTTP (не трогаем /metrics и health endpoints)
	r.Use(metricsMiddleware())

	// Prometheus endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health endpoints
	r.GET("/healthz", healthzHandler)
	r.GET("/readiness", readinessHandler)

	// Основные endpoints
	r.Any("/invoke", invokeHandler)
	r.Any("/api/v1/metrics", invokeHandler) // Совместимость с echo-примером

	log.Printf("Waiter service starting on :8080")
	log.Printf("Service: %s, Revision: %s, Tenant: %s", serviceName, revision, tenantID)

	// ВАЖНО: фиксированный порт 8080, чтобы совпадать с containerPort в YAML
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return gin.LoggerWithWriter(gin.DefaultWriter, "/metrics", "/healthz", "/readiness")
}

func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(startTime).String(),
	})
}

func readinessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"memory": "ok",
			"disk":   "ok",
		},
	})
}

func invokeHandler(c *gin.Context) {
	start := time.Now()
	labels := []string{c.Request.Method, c.FullPath(), tenantID, serviceName, revision}

	var req WaiterRequest
	if err := c.ShouldBind(&req); err != nil {
		requestsTotal.WithLabelValues(append(labels, "400")...).Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Значения по умолчанию
	if req.SleepMs == 0 {
		req.SleepMs = 100
	}
	if req.MemMB == 0 {
		req.MemMB = 10
	}

	// Простой критерий cold start (первые 30s после старта pod)
	isColdStart := time.Since(startTime) < 30*time.Second

	// Эмуляция выделения памяти
	var memBuf []byte
	if req.MemMB > 0 {
		memBuf = make([]byte, req.MemMB*1024*1024)
		for i := 0; i < len(memBuf); i += 4096 {
			memBuf[i] = byte(i % 256)
		}
	}

	// CPU нагрузка
	if req.CpuSpinMs > 0 {
		cpuStart := time.Now()
		for time.Since(cpuStart).Milliseconds() < int64(req.CpuSpinMs) {
			_ = fibonacci(35)
		}
	}

	// Искусственная задержка (I/O)
	if req.SleepMs > 0 {
		time.Sleep(time.Duration(req.SleepMs) * time.Millisecond)
	}

	duration := time.Since(start)

	// Метрики
	requestsTotal.WithLabelValues(append(labels, "200")...).Inc()
	requestDuration.WithLabelValues(labels[:len(labels)]...).Observe(duration.Seconds())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage.WithLabelValues(tenantID, serviceName, revision).Set(float64(m.Alloc))

	resp := WaiterResponse{
		Request:       req,
		DurationMs:    duration.Milliseconds(),
		MemoryAllocMB: float64(m.Alloc) / 1024.0 / 1024.0,
		ColdStart:     isColdStart,
		Timestamp:     time.Now().Unix(),
		Metadata: Metadata{
			ServiceName: serviceName,
			Revision:    revision,
			TenantID:    tenantID,
			PodName:     getEnv("HOSTNAME", "unknown"),
			Uptime:      time.Since(startTime).String(),
		},
	}

	c.JSON(http.StatusOK, resp)
}

// Простая CPU-интенсивная функция
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
