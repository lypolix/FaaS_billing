package main

import (
	"context"
	"encoding/json"
	"time"
)

type IngestRequest []UsageRaw

type BillingCalcRequest struct {
	TenantID  string    `json:"tenant_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	ServiceID string    `json:"service_id,omitempty"`
}

type Agent struct {
	cfg     Config
	log     *Logger
	tx      *Transport
	source  MetricSource
	batchSz int
}

func NewAgent(cfg Config, log *Logger, tx *Transport, src MetricSource) *Agent {
	return &Agent{
		cfg:     cfg,
		log:     log,
		tx:      tx,
		source:  src,
		batchSz: cfg.BatchSize,
	}
}

func (a *Agent) runOnce(ctx context.Context) error {
	metrics, err := a.source.Collect(ctx)
	if err != nil {
		a.log.Error("collect failed", map[string]any{"err": err.Error()})
		return err
	}
	if len(metrics) == 0 {
		a.log.Info("no metrics", nil)
		return nil
	}
	// батчинг
	for i := 0; i < len(metrics); i += a.batchSz {
		j := i + a.batchSz
		if j > len(metrics) {
			j = len(metrics)
		}
		batch := IngestRequest(metrics[i:j])
		if err := a.pushBatch(ctx, batch); err != nil {
			return err
		}
	}

	if a.cfg.CalcOnPush {
		end := time.Now().UTC().Truncate(time.Minute)
		start := end.Add(-a.cfg.CalcWindow)
		req := BillingCalcRequest{
			TenantID:  a.cfg.TenantID,
			StartTime: start,
			EndTime:   end,
		}
		if a.cfg.CalcServiceOnly {
			req.ServiceID = a.cfg.ServiceID
		}
		if err := a.calc(ctx, req); err != nil {
			a.log.Warn("billing calc failed", map[string]any{"err": err.Error()})
		}
	}
	return nil
}

func (a *Agent) pushBatch(ctx context.Context, batch IngestRequest) error {
	resp, err := a.tx.postJSON(ctx, "/metrics/ingest", batch)
	if err != nil {
		a.log.Error("ingest request failed", map[string]any{"err": err.Error()})
		return err
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	a.log.Info("ingest ok", out)
	return nil
}

func (a *Agent) calc(ctx context.Context, req BillingCalcRequest) error {
	resp, err := a.tx.postJSON(ctx, "/billing/calculate", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	a.log.Info("calc ok", out)
	return nil
}

func (a *Agent) Run(ctx context.Context) {
	tick := time.NewTicker(a.cfg.PushInterval)
	defer tick.Stop()
	a.log.Info("agent started", map[string]any{
		"interval": a.cfg.PushInterval.String(),
	})
	// первый запуск сразу
	_ = a.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			a.log.Info("agent stopped", nil)
			return
		case <-tick.C:
			_ = a.runOnce(ctx)
		}
	}
}
