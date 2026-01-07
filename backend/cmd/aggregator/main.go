package main

import (
	"flag"
	"log"
	"time"

	"github.com/lypolix/FaaS-billing/internal/database"
	"github.com/lypolix/FaaS-billing/internal/models"
)

type aggRow struct {
	TenantID   string
	ServiceID  string
	RevisionID *string

	Invocations     int64
	TotalDurationMS int64
	AvgDurationMS   float64

	MaxMemoryMB float64
	AvgMemoryMB float64

	ColdStarts  int64
	Errors      int64
	EgressBytes int64

	// Sum(memory_mb) * window_seconds / 3600
	TotalMemoryMBHours float64
}

func main() {
	var (
		windowStr string
		endStr    string
	)
	flag.StringVar(&windowStr, "window", "1m", "Aggregation window size: 1m,5m,1h,1d")
	flag.StringVar(&endStr, "end", "", "Optional window end time in RFC3339 (UTC recommended). Example: 2026-01-07T12:00:00Z")
	flag.Parse()

	window, err := time.ParseDuration(windowStr)
	if err != nil {
		log.Fatalf("invalid -window: %v", err)
	}

	var end time.Time
	if endStr == "" {
		end = time.Now().UTC().Truncate(window)
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			log.Fatalf("invalid -end: %v", err)
		}
		end = end.UTC().Truncate(window)
	}
	start := end.Add(-window)

	database.Connect()

	log.Printf("aggregate: window=%s start=%s end=%s", windowStr, start.Format(time.RFC3339), end.Format(time.RFC3339))

	rows, err := queryAggRows(start, end, window)
	if err != nil {
		log.Fatalf("query aggregates: %v", err)
	}

	if len(rows) == 0 {
		log.Printf("no usage_raws rows in window, nothing to aggregate")
		return
	}

	if err := upsertAggregates(start, end, windowStr, rows); err != nil {
		log.Fatalf("upsert aggregates: %v", err)
	}

	log.Printf("done: upserted=%d", len(rows))
}

func queryAggRows(start, end time.Time, window time.Duration) ([]aggRow, error) {
	windowSeconds := window.Seconds()

	q := `
			SELECT
			tenant_id::text AS tenant_id,
			service_id::text AS service_id,
			revision_id::text AS revision_id,

			COALESCE(SUM(CASE WHEN metric_name = 'invocations' THEN value ELSE 0 END), 0)::bigint AS invocations,
			COALESCE(SUM(CASE WHEN metric_name = 'duration_ms' THEN value ELSE 0 END), 0)::bigint AS total_duration_ms,

			COALESCE(AVG(CASE WHEN metric_name = 'duration_ms' THEN value END), 0)::float8 AS avg_duration_ms,

			COALESCE(MAX(CASE WHEN metric_name = 'memory_mb' THEN value END), 0)::float8 AS max_memory_mb,
			COALESCE(AVG(CASE WHEN metric_name = 'memory_mb' THEN value END), 0)::float8 AS avg_memory_mb,

			COALESCE(SUM(CASE WHEN metric_name = 'cold_starts' THEN value ELSE 0 END), 0)::bigint AS cold_starts,
			COALESCE(SUM(CASE WHEN metric_name = 'errors' THEN value ELSE 0 END), 0)::bigint AS errors,
			COALESCE(SUM(CASE WHEN metric_name = 'egress_bytes' THEN value ELSE 0 END), 0)::bigint AS egress_bytes,

			-- MB * hour for this window:
			(COALESCE(SUM(CASE WHEN metric_name = 'memory_mb' THEN value ELSE 0 END), 0) * $3 / 3600.0)::float8 AS total_memory_mb_hours
			FROM usage_raws
			WHERE timestamp >= $1 AND timestamp < $2
			GROUP BY tenant_id, service_id, revision_id;
		`

	var out []aggRow
	if err := database.DB.Raw(q, start, end, windowSeconds).Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func upsertAggregates(start, end time.Time, windowSize string, rows []aggRow) error {
	ins := `
		INSERT INTO usage_aggregates (
		window_start, window_end, window_size,
		tenant_id, service_id, revision_id,
		invocations, total_duration_ms, avg_duration_ms, p50_duration_ms, p95_duration_ms,
		max_memory_mb, avg_memory_mb, total_memory_mb_hours,
		cold_starts, errors, egress_bytes
		)
		VALUES (
		$1, $2, $3,
		$4::uuid, $5::uuid, $6::uuid,
		$7, $8, $9, 0, 0,
		$10, $11, $12,
		$13, $14, $15
		)
		ON CONFLICT (window_start, window_end, tenant_id, service_id, revision_id)
		DO UPDATE SET
		invocations = EXCLUDED.invocations,
		total_duration_ms = EXCLUDED.total_duration_ms,
		avg_duration_ms = EXCLUDED.avg_duration_ms,
		max_memory_mb = EXCLUDED.max_memory_mb,
		avg_memory_mb = EXCLUDED.avg_memory_mb,
		total_memory_mb_hours = EXCLUDED.total_memory_mb_hours,
		cold_starts = EXCLUDED.cold_starts,
		errors = EXCLUDED.errors,
		egress_bytes = EXCLUDED.egress_bytes,
		window_size = EXCLUDED.window_size;
	`

	insNullRev := `
		INSERT INTO usage_aggregates (
		window_start, window_end, window_size,
		tenant_id, service_id, revision_id,
		invocations, total_duration_ms, avg_duration_ms, p50_duration_ms, p95_duration_ms,
		max_memory_mb, avg_memory_mb, total_memory_mb_hours,
		cold_starts, errors, egress_bytes
		)
		VALUES (
		$1, $2, $3,
		$4::uuid, $5::uuid, NULL,
		$6, $7, $8, 0, 0,
		$9, $10, $11,
		$12, $13, $14
		)
		ON CONFLICT (window_start, window_end, tenant_id, service_id, revision_id)
		DO UPDATE SET
		invocations = EXCLUDED.invocations,
		total_duration_ms = EXCLUDED.total_duration_ms,
		avg_duration_ms = EXCLUDED.avg_duration_ms,
		max_memory_mb = EXCLUDED.max_memory_mb,
		avg_memory_mb = EXCLUDED.avg_memory_mb,
		total_memory_mb_hours = EXCLUDED.total_memory_mb_hours,
		cold_starts = EXCLUDED.cold_starts,
		errors = EXCLUDED.errors,
		egress_bytes = EXCLUDED.egress_bytes,
		window_size = EXCLUDED.window_size;
`

	tx := database.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	for _, r := range rows {
		if r.RevisionID == nil || *r.RevisionID == "" {
			if err := tx.Exec(insNullRev,
				start, end, windowSize,
				r.TenantID, r.ServiceID,
				r.Invocations,
				r.TotalDurationMS,
				r.AvgDurationMS,
				r.MaxMemoryMB,
				r.AvgMemoryMB,
				r.TotalMemoryMBHours,
				r.ColdStarts,
				r.Errors,
				r.EgressBytes,
			).Error; err != nil {
				tx.Rollback()
				return err
			}
			continue
		}

		if err := tx.Exec(ins,
			start, end, windowSize,
			r.TenantID, r.ServiceID, *r.RevisionID,
			r.Invocations,
			r.TotalDurationMS,
			r.AvgDurationMS,
			r.MaxMemoryMB,
			r.AvgMemoryMB,
			r.TotalMemoryMBHours,
			r.ColdStarts,
			r.Errors,
			r.EgressBytes,
		).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	
	tx.Where("timestamp >= ? AND timestamp < ?", start, end).Delete(&models.UsageRaw{})
	
	return tx.Commit().Error
}
