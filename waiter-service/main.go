package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Метрики для Prometheus
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waiter_requests_total",
			Help: "Total number of requests handled by the waiter service",
		},
		[]string{"method", "endpoint", "status", "tenant_id", "service_name", "revision"},
	)
	
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "waiter_request_duration_seconds",
			Help:    "Duration of requests",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"tenant_id", "service_name", "revision"},
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
	
	egressBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "waiter_egress_bytes_total",
			Help: "Total egress traffic in bytes",
		},
		[]string{"tenant_id", "service_name", "revision"},
	)

	startTime = time.Now()
	coldStartDetected = false
	mu sync.Mutex
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, memoryUsage, coldStarts, egressBytes)
}

func main() {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	
	// Метки для метрик (из env или дефолтные)
	tenantID := getEnv("TENANT_ID", "demo-tenant")
	serviceName := getEnv("SERVICE_NAME", "waiter")
	revision := getEnv("REVISION", "waiter-00001")
	
	// Детектируем холодный старт (первый запрос после запуска)
	r.Use(func(c *gin.Context) {
		mu.Lock()
		if !coldStartDetected && time.Since(startTime) > 1*time.Second {
			coldStarts.WithLabelValues(tenantID, serviceName, revision).Inc()
			coldStartDetected = true
		}
		mu.Unlock()
		c.Next()
	})

	// Основной эндпоинт для имитации нагрузки
	r.GET("/invoke", func(c *gin.Context) {
		start := time.Now()
		
		// Парсим параметры
		sleepMS, _ := strconv.Atoi(c.DefaultQuery("sleep_ms", "0"))
		memMB, _ := strconv.Atoi(c.DefaultQuery("mem_mb", "0"))
		cpuSpinMS, _ := strconv.Atoi(c.DefaultQuery("cpu_spin_ms", "0"))
		generateEgress := c.DefaultQuery("egress", "false") == "true"
		
		// Имитируем задержку I/O
		if sleepMS > 0 {
			time.Sleep(time.Duration(sleepMS) * time.Millisecond)
		}
		
		// Имитируем потребление памяти
		var memBallast []byte
		if memMB > 0 {
			memBallast = make([]byte, memMB*1024*1024)
			for i := range memBallast {
				memBallast[i] = byte(rand.Intn(256))
			}
		}
		
		// Имитируем CPU нагрузку
		if cpuSpinMS > 0 {
			cpuEnd := time.Now().Add(time.Duration(cpuSpinMS) * time.Millisecond)
			for time.Now().Before(cpuEnd) {
				_ = rand.Int() // CPU-интенсивная операция
			}
		}
		
		// Обновляем метрики памяти
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memoryUsage.WithLabelValues(tenantID, serviceName, revision).Set(float64(m.Alloc))
		
		duration := time.Since(start)
		
		// Обновляем метрики
		requestsTotal.WithLabelValues("GET", "/invoke", "200", tenantID, serviceName, revision).Inc()
		requestDuration.WithLabelValues(tenantID, serviceName, revision).Observe(duration.Seconds())
		
		response := map[string]interface{}{
			"timestamp":       time.Now().UTC(),
			"duration_ms":     duration.Milliseconds(),
			"memory_alloc_mb": float64(m.Alloc) / (1024 * 1024),
			"memory_sys_mb":   float64(m.Sys) / (1024 * 1024),
			"cold_start":      !coldStartDetected && time.Since(startTime) <= 5*time.Second,
			"parameters": map[string]interface{}{
				"sleep_ms":    sleepMS,
				"mem_mb":      memMB,
				"cpu_spin_ms": cpuSpinMS,
				"egress":      generateEgress,
			},
			"tenant_id":    tenantID,
			"service_name": serviceName,
			"revision":     revision,
		}
		
		// Имитируем исходящий трафик
		responseJSON, _ := json.Marshal(response)
		responseSize := len(responseJSON)
		
		if generateEgress {
			// Добавляем фиктивный payload для имитации трафика
			response["payload"] = generateRandomString(10 * 1024) // 10KB
			responseJSON, _ = json.Marshal(response)
			responseSize = len(responseJSON)
		}
		
		// Обновляем метрику исходящего трафика
		egressBytes.WithLabelValues(tenantID, serviceName, revision).Add(float64(responseSize))
		
		c.JSON(200, response)
	})

	// Prometheus метрики
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	// Health checks
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "uptime": time.Since(startTime).String()})
	})
	
	r.GET("/readiness", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready", "timestamp": time.Now().UTC()})
	})

	// Запуск сервера
	port := getEnv("PORT", "8080")
	log.Printf("Starting waiter service on port %s", port)
	log.Printf("Tenant: %s, Service: %s, Revision: %s", tenantID, serviceName, revision)
	
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}
