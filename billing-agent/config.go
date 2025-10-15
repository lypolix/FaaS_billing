package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BackendURL      string        // http://localhost:8081/api/v1
	TenantID        string        // UUID
	ServiceID       string        // UUID
	RevisionID      string        // UUID (optional)
	BatchSize       int           // metrics batch size
	PushInterval    time.Duration // how often to push metrics
	HttpTimeout     time.Duration // single request timeout
	Retries         int           // retry attempts
	RetryBackoff    time.Duration // between retries
	CalcOnPush      bool          // call /billing/calculate after push
	CalcWindow      time.Duration // billing window (e.g. 1h)
	CalcServiceOnly bool          // pass service_id to calculation
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func LoadConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.BackendURL, "backend", getenv("BA_BACKEND", "http://localhost:8081/api/v1"), "Backend base URL (…/api/v1)")
	flag.StringVar(&cfg.TenantID, "tenant", getenv("BA_TENANT_ID", ""), "Tenant UUID")
	flag.StringVar(&cfg.ServiceID, "service", getenv("BA_SERVICE_ID", ""), "Service UUID")
	flag.StringVar(&cfg.RevisionID, "revision", getenv("BA_REVISION_ID", ""), "Revision UUID (optional)")
	flag.IntVar(&cfg.BatchSize, "batch", getEnvInt("BA_BATCH", 200), "Batch size for ingest")
	flag.DurationVar(&cfg.PushInterval, "interval", getEnvDuration("BA_INTERVAL", time.Minute), "Push interval")
	flag.DurationVar(&cfg.HttpTimeout, "timeout", getEnvDuration("BA_HTTP_TIMEOUT", 10*time.Second), "HTTP timeout")
	flag.IntVar(&cfg.Retries, "retries", getEnvInt("BA_RETRIES", 3), "HTTP retries")
	flag.DurationVar(&cfg.RetryBackoff, "backoff", getEnvDuration("BA_RETRY_BACKOFF", 1*time.Second), "Retry backoff")
	flag.BoolVar(&cfg.CalcOnPush, "calc", getEnvBool("BA_CALC", false), "Run billing calculation after push")
	flag.DurationVar(&cfg.CalcWindow, "calc-window", getEnvDuration("BA_CALC_WINDOW", time.Hour), "Billing window duration")
	flag.BoolVar(&cfg.CalcServiceOnly, "calc-service-only", getEnvBool("BA_CALC_SERVICE_ONLY", true), "Pass service_id to billing")
	flag.Parse()
	return cfg
}


func getEnvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getEnvBool(k string, def bool) bool {
	v := strings.ToLower(os.Getenv(k))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

