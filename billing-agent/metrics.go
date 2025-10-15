package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UsageRaw совместим c backend’ом
type UsageRaw struct {
	TenantID   uuid.UUID  `json:"tenant_id"`
	ServiceID  uuid.UUID  `json:"service_id"`
	RevisionID *uuid.UUID `json:"revision_id,omitempty"`

	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Temporality string    `json:"temporality"`
	Timestamp   time.Time `json:"timestamp"`
	Labels      string    `json:"labels"`
}

// Генерация метрик (пример: mock/телеметрия процесса)
type MetricSource interface {
	Collect(ctx context.Context) ([]UsageRaw, error)
}

type ProcessMetrics struct {
	tenant uuid.UUID
	svc    uuid.UUID
	rev    *uuid.UUID
}

func NewProcessMetrics(tenant, svc, rev string) (*ProcessMetrics, error) {
	tid, err := uuid.Parse(tenant)
	if err != nil {
		return nil, err
	}
	sid, err := uuid.Parse(svc)
	if err != nil {
		return nil, err
	}
	var rid *uuid.UUID
	if rev != "" {
		tmp, err := uuid.Parse(rev)
		if err != nil {
			return nil, err
		}
		rid = &tmp
	}
	return &ProcessMetrics{tenant: tid, svc: sid, rev: rid}, nil
}

func (p *ProcessMetrics) Collect(ctx context.Context) ([]UsageRaw, error) {
	now := time.Now().UTC()
	// Тут можно подставить реальные показатели (из экспортеров/SDK/Knative/Prometheus).
	// Пока — демо-значения.
	return []UsageRaw{
		{
			TenantID:   p.tenant,
			ServiceID:  p.svc,
			RevisionID: p.rev,
			Metric:     "invocations",
			Value:      150,
			Unit:       "count",
			Temporality:"delta",
			Timestamp:  now,
			Labels:     `{"source":"agent"}`,
		},
		{
			TenantID:   p.tenant,
			ServiceID:  p.svc,
			RevisionID: p.rev,
			Metric:     "memory_mbms",
			Value:      32000,
			Unit:       "MB-ms",
			Temporality:"delta",
			Timestamp:  now,
			Labels:     `{"source":"agent"}`,
		},
		{
			TenantID:   p.tenant,
			ServiceID:  p.svc,
			RevisionID: p.rev,
			Metric:     "cold_starts",
			Value:      1,
			Unit:       "count",
			Temporality:"delta",
			Timestamp:  now,
			Labels:     `{"source":"agent"}`,
		},
	}, nil
}
