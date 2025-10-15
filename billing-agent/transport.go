package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Transport struct {
	baseURL string
	hc      *http.Client
	retries int
	backoff time.Duration
	log     *Logger
}

func NewTransport(cfg Config, log *Logger) *Transport {
	dialer := &net.Dialer{Timeout: cfg.HttpTimeout}
	tr := &http.Transport{
		DialContext:         dialer.DialContext,
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &Transport{
		baseURL: cfg.BackendURL,
		hc:      &http.Client{Timeout: cfg.HttpTimeout, Transport: tr},
		retries: cfg.Retries,
		backoff: cfg.RetryBackoff,
		log:     log,
	}
}

func (t *Transport) postJSON(ctx context.Context, path string, body any) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", t.baseURL, path)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt <= t.retries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.hc.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		lastErr = err
		time.Sleep(t.backoff * time.Duration(attempt+1))
	}
	if lastErr == nil {
		lastErr = errors.New("postJSON retries exceeded")
	}
	return nil, lastErr
}
