package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	TenantID    string            `json:"tenant_id"`
	ServiceName string            `json:"service_name"`
	Revision    string            `json:"revision"`
	Invocations int64             `json:"invocations"`
	Duration    float64           `json:"duration_seconds"`
	MemoryMB    float64           `json:"memory_mb"`
	ColdStart   bool              `json:"cold_start"`
	Labels      map[string]string `json:"labels,omitempty"`
}

var (
	rdb       *redis.Client
	queueKey  string
	startTime = time.Now()

	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "queue_http_requests_total", Help: "HTTP requests"},
		[]string{"method", "path", "status"},
	)
	ingestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "queue_ingest_total", Help: "Ingested events"},
		[]string{"tenant_id", "service_name", "revision"},
	)
	ingestErr = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "queue_ingest_errors_total", Help: "Ingest errors"},
		[]string{"reason"},
	)
	ingestDur = prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: "queue_ingest_duration_seconds", Help: "Ingest latency", Buckets: []float64{0.005, 0.02, 0.1, 0.3, 1, 2, 5}},
	)
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	prometheus.MustRegister(reqTotal, ingestTotal, ingestErr, ingestDur)

	rdb = redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "redis:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	queueKey = getEnv("REDIS_QUEUE_KEY", "metrics_queue")

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		reqTotal.WithLabelValues(c.Request.Method, c.FullPath(), http.StatusText(c.Writer.Status())).Inc()
	})

	// Health/metrics
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "uptime": time.Since(startTime).String()})
	})
	r.GET("/readiness", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Приём событий (один или батч)
	r.POST("/metrics/collect", func(c *gin.Context) {
		begin := time.Now()
		defer func() { ingestDur.Observe(time.Since(begin).Seconds()) }()

		var arr []MetricEvent
		if err := c.ShouldBindJSON(&arr); err != nil {
			var one MetricEvent
			if err2 := c.ShouldBindJSON(&one); err2 != nil {
				ingestErr.WithLabelValues("decode").Inc()
				c.JSON(400, gin.H{"error": "invalid JSON"})
				return
			}
			arr = []MetricEvent{one}
		}

		ctx := context.Background()
		accepted := 0
		for _, ev := range arr {
			if ev.Timestamp.IsZero() {
				ev.Timestamp = time.Now().UTC()
			}
			if ev.TenantID == "" {
				ev.TenantID = getEnv("DEFAULT_TENANT", "demo-tenant")
			}
			if ev.ServiceName == "" {
				ev.ServiceName = getEnv("DEFAULT_SERVICE", "waiter")
			}
			if ev.Revision == "" {
				ev.Revision = getEnv("DEFAULT_REVISION", "rev-unknown")
			}
			b, err := json.Marshal(ev)
			if err != nil {
				ingestErr.WithLabelValues("marshal").Inc()
				continue
			}
			if err := rdb.LPush(ctx, queueKey, b).Err(); err != nil {
				ingestErr.WithLabelValues("redis_lpush").Inc()
				continue
			}
			ingestTotal.WithLabelValues(ev.TenantID, ev.ServiceName, ev.Revision).Inc()
			accepted++
		}
		c.JSON(200, gin.H{"accepted": accepted, "total": len(arr)})
	})

	// Выдача события потребителю (по одному)
	r.POST("/metrics/pop", func(c *gin.Context) {
		ctx := context.Background()
		res := rdb.RPop(ctx, queueKey)
		if res.Err() != nil {
			c.JSON(204, gin.H{"message": "empty"})
			return
		}
		c.Data(200, "application/json", []byte(res.Val()))
	})

	log.Println("queue starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
