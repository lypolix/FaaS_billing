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

var (
	ingestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queueproxy_ingest_total",
			Help: "Total number of ingested metric events",
		},
		[]string{"tenant_id", "service_name", "revision"},
	)
	ingestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queueproxy_ingest_errors_total",
			Help: "Total number of ingest errors",
		},
		[]string{"reason"},
	)
	ingestLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "queueproxy_ingest_duration_seconds",
			Help:    "Ingest handler duration",
			Buckets: []float64{0.005, 0.02, 0.1, 0.3, 1, 2, 5},
		},
	)
	startTime = time.Now()
)

type MetricEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	ServiceName string    `json:"service_name"`
	Revision    string    `json:"revision"`
	TenantID    string    `json:"tenant_id"`

	Invocations int64   `json:"invocations"`
	Duration    float64 `json:"duration_seconds"`
	MemoryMB    float64 `json:"memory_mb"`
	ColdStart   bool    `json:"cold_start"`
	Labels      map[string]string `json:"labels,omitempty"`
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	prometheus.MustRegister(ingestTotal, ingestErrors, ingestLatency)

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"uptime":  time.Since(startTime).String(),
			"version": "v1",
		})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "redis:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	// Ingest endpoint: принимает массив событий или одно событие
	r.POST("/metrics/collect", func(c *gin.Context) {
		begin := time.Now()
		defer func() { ingestLatency.Observe(time.Since(begin).Seconds()) }()

		dec := json.NewDecoder(c.Request.Body)
		dec.DisallowUnknownFields()

		// Пытаемся распарсить массив
		var batch []MetricEvent
		if err := dec.Decode(&batch); err != nil {
			// если не массив — пробуем одиночное событие
			c.Request.Body.Close()
			if c.Request.ContentLength == 0 {
				ingestErrors.WithLabelValues("empty_body").Inc()
				c.JSON(http.StatusBadRequest, gin.H{"error": "empty body"})
				return
			}
			// читаем тело заново
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
			if err := c.ShouldBindJSON(&batch); err != nil {
				var one MetricEvent
				if err2 := c.ShouldBindJSON(&one); err2 != nil {
					ingestErrors.WithLabelValues("decode").Inc()
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
					return
				}
				batch = []MetricEvent{one}
			}
		}

		ctx := context.Background()
		pushed := 0
		for _, ev := range batch {
			// sane defaults
			if ev.Timestamp.IsZero() {
				ev.Timestamp = time.Now().UTC()
			}
			if ev.TenantID == "" {
				ev.TenantID = getEnv("DEFAULT_TENANT", "demo-tenant")
			}
			if ev.ServiceName == "" {
				ev.ServiceName = getEnv("DEFAULT_SERVICE", "unknown-service")
			}
			if ev.Revision == "" {
				ev.Revision = getEnv("DEFAULT_REVISION", "rev-unknown")
			}

			data, err := json.Marshal(ev)
			if err != nil {
				ingestErrors.WithLabelValues("marshal").Inc()
				continue
			}
			if err := rdb.LPush(ctx, getEnv("REDIS_QUEUE_KEY", "metrics_queue"), data).Err(); err != nil {
				ingestErrors.WithLabelValues("redis_lpush").Inc()
				continue
			}
			ingestTotal.WithLabelValues(ev.TenantID, ev.ServiceName, ev.Revision).Inc()
			pushed++
		}

		c.JSON(http.StatusOK, gin.H{
			"accepted": pushed,
			"total":    len(batch),
		})
	})

	log.Println("queue-proxy starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
