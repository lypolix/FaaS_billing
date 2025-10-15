package services

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/lypolix/FaaS-billing/internal/models"
	"gorm.io/gorm"
)

type MetricsService struct {
	db *gorm.DB
}

func NewMetricsService(db *gorm.DB) *MetricsService {
	return &MetricsService{db: db}
}

// Приём сырого батча метрик
func (s *MetricsService) IngestMetrics(batch []models.UsageRaw) error {
	if len(batch) == 0 {
		return nil
	}
	return s.db.Create(&batch).Error
}

func (s *MetricsService) AggregateMetrics(startTime, endTime time.Time, windowSize string) error {
	windowDuration, err := parseWindowSize(windowSize)
	if err != nil {
		return err
	}
	current := startTime.Truncate(windowDuration)
	for current.Before(endTime) {
		if err := s.aggregateWindow(current, current.Add(windowDuration), windowSize); err != nil {
			return err
		}
		current = current.Add(windowDuration)
	}
	return nil
}

// ключи читаем строками (как пришли из БД), далее приводим к uuid
type aggregateKey struct {
	TenantID   string
	ServiceID  string
	RevisionID *string
}

func (s *MetricsService) aggregateWindow(windowStart, windowEnd time.Time, windowSize string) error {
	var keys []aggregateKey
	if err := s.db.Model(&models.UsageRaw{}).
		Select("DISTINCT tenant_id, service_id, revision_id").
		Where("timestamp >= ? AND timestamp < ?", windowStart, windowEnd).
		Scan(&keys).Error; err != nil {
		return err
	}

	for _, k := range keys {
		if err := s.aggregateForKey(k, windowStart, windowEnd, windowSize); err != nil {
			return err
		}
	}
	return nil
}

func (s *MetricsService) aggregateForKey(k aggregateKey, windowStart, windowEnd time.Time, windowSize string) error {
	// Приведение типов string -> uuid.UUID
	tenantUUID, err := uuid.Parse(k.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant_id uuid: %s: %w", k.TenantID, err)
	}
	serviceUUID, err := uuid.Parse(k.ServiceID)
	if err != nil {
		return fmt.Errorf("invalid service_id uuid: %s: %w", k.ServiceID, err)
	}
	var revisionUUIDPtr *uuid.UUID
	if k.RevisionID != nil && *k.RevisionID != "" {
		if rev, err := uuid.Parse(*k.RevisionID); err == nil {
			revisionUUIDPtr = &rev
		} else {
			return fmt.Errorf("invalid revision_id uuid: %s: %w", *k.RevisionID, err)
		}
	}

	// пропускаем, если уже есть запись
	var exists int64
	s.db.Model(&models.UsageAggregate{}).
		Where("tenant_id = ? AND service_id = ? AND window_start = ? AND window_size = ?",
			tenantUUID, serviceUUID, windowStart, windowSize).
		Count(&exists)
	if exists > 0 {
		return nil
	}

	agg := models.UsageAggregate{
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		WindowSize:  windowSize,
		TenantID:    tenantUUID,
		ServiceID:   serviceUUID,
		RevisionID:  revisionUUIDPtr,
	}

	// invocations
	var invocations sql.NullFloat64
	q := s.db.Model(&models.UsageRaw{}).Where(
		"tenant_id = ? AND service_id = ? AND timestamp >= ? AND timestamp < ? AND metric_name = ?",
		tenantUUID, serviceUUID, windowStart, windowEnd, "invocations",
	)
	if revisionUUIDPtr != nil {
		q = q.Where("revision_id = ?", *revisionUUIDPtr)
	} else {
		q = q.Where("revision_id IS NULL")
	}
	q.Select("SUM(value)").Scan(&invocations)
	if invocations.Valid {
		agg.Invocations = int64(invocations.Float64)
	}

	// duration
	var durationSum, durationCount sql.NullFloat64
	q = s.db.Model(&models.UsageRaw{}).Where(
		"tenant_id = ? AND service_id = ? AND timestamp >= ? AND timestamp < ? AND metric_name = ?",
		tenantUUID, serviceUUID, windowStart, windowEnd, "duration_ms",
	)
	if revisionUUIDPtr != nil {
		q = q.Where("revision_id = ?", *revisionUUIDPtr)
	} else {
		q = q.Where("revision_id IS NULL")
	}
	q.Select("SUM(value)").Scan(&durationSum)
	q.Select("COUNT(*)").Scan(&durationCount)
	if durationSum.Valid && durationCount.Valid && durationCount.Float64 > 0 {
		agg.TotalDurationMS = int64(durationSum.Float64)
		agg.AvgDurationMS = durationSum.Float64 / durationCount.Float64

		// p50/p95 в памяти
		var durations []float64
		s.db.Model(&models.UsageRaw{}).
			Where(
				"tenant_id = ? AND service_id = ? AND timestamp >= ? AND timestamp < ? AND metric_name = ?",
				tenantUUID, serviceUUID, windowStart, windowEnd, "duration_ms",
			).
			Pluck("value", &durations)
		if len(durations) > 0 {
			sort.Float64s(durations)
			agg.P50DurationMS = percentile(durations, 50)
			agg.P95DurationMS = percentile(durations, 95)
		}
	}

	// memory: avg/max + MB*hours
	var memorySum, memoryMax, memoryCount sql.NullFloat64
	q = s.db.Model(&models.UsageRaw{}).Where(
		"tenant_id = ? AND service_id = ? AND timestamp >= ? AND timestamp < ? AND metric_name = ?",
		tenantUUID, serviceUUID, windowStart, windowEnd, "memory_mb",
	)
	if revisionUUIDPtr != nil {
		q = q.Where("revision_id = ?", *revisionUUIDPtr)
	} else {
		q = q.Where("revision_id IS NULL")
	}
	q.Select("SUM(value)").Scan(&memorySum)
	q.Select("MAX(value)").Scan(&memoryMax)
	q.Select("COUNT(*)").Scan(&memoryCount)
	if memorySum.Valid && memoryCount.Valid && memoryCount.Float64 > 0 {
		agg.AvgMemoryMB = memorySum.Float64 / memoryCount.Float64
	}
	if memoryMax.Valid {
		agg.MaxMemoryMB = memoryMax.Float64
	}
	agg.TotalMemoryMBHours = agg.AvgMemoryMB * windowEnd.Sub(windowStart).Hours()

	// cold_starts
	var coldStarts sql.NullFloat64
	q = s.db.Model(&models.UsageRaw{}).Where(
		"tenant_id = ? AND service_id = ? AND timestamp >= ? AND timestamp < ? AND metric_name = ?",
		tenantUUID, serviceUUID, windowStart, windowEnd, "cold_starts",
	)
	if revisionUUIDPtr != nil {
		q = q.Where("revision_id = ?", *revisionUUIDPtr)
	} else {
		q = q.Where("revision_id IS NULL")
	}
	q.Select("SUM(value)").Scan(&coldStarts)
	if coldStarts.Valid {
		agg.ColdStarts = int(coldStarts.Float64)
	}

	// egress
	var egressSum sql.NullFloat64
	q = s.db.Model(&models.UsageRaw{}).Where(
		"tenant_id = ? AND service_id = ? AND timestamp >= ? AND timestamp < ? AND metric_name = ?",
		tenantUUID, serviceUUID, windowStart, windowEnd, "egress_bytes",
	)
	if revisionUUIDPtr != nil {
		q = q.Where("revision_id = ?", *revisionUUIDPtr)
	} else {
		q = q.Where("revision_id IS NULL")
	}
	q.Select("SUM(value)").Scan(&egressSum)
	if egressSum.Valid {
		agg.EgressBytes = int64(egressSum.Float64)
	}

	return s.db.Create(&agg).Error
}

func parseWindowSize(s string) (time.Duration, error) {
	switch s {
	case "1m":
		return time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported window size: %s", s)
	}
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := float64(p) / 100.0 * float64(len(sorted)-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[lo]
	}
	f := idx - float64(lo)
	return sorted[lo]*(1-f) + sorted[hi]*f
}
