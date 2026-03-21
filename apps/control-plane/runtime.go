package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type proxyRuntimeConfig struct {
	Mode          string            `json:"mode"`
	Primary       string            `json:"primary"`
	PrimaryOrigin string            `json:"primary_origin"`
	Origins       map[string]string `json:"origins"`
}

type kernelHealth struct {
	Name       string    `json:"name"`
	Origin     string    `json:"origin"`
	Configured bool      `json:"configured"`
	Healthy    bool      `json:"healthy"`
	StatusCode int       `json:"status_code,omitempty"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checked_at,omitempty"`
	Role       string    `json:"role,omitempty"`
}

type kernelStatusResponse struct {
	Mode      string                  `json:"mode"`
	Primary   string                  `json:"primary"`
	Kernels   map[string]kernelHealth `json:"kernels"`
	CheckedAt time.Time               `json:"checked_at"`
}

func loadProxyRuntimeConfig() proxyRuntimeConfig {
	mode := strings.ToLower(strings.TrimSpace(getenv("PROXY_CORE_MODE", "go")))
	if mode != "go" && mode != "rust" && mode != "dual" {
		mode = "go"
	}

	legacyOrigin := strings.TrimSpace(getenv("PROXY_CORE_ORIGIN", "http://127.0.0.1:8081"))
	goOrigin := strings.TrimSpace(getenv("PROXY_CORE_GO_ORIGIN", legacyOrigin))
	rustOrigin := strings.TrimSpace(getenv("PROXY_CORE_RUST_ORIGIN", "http://127.0.0.1:8045"))
	primary := strings.ToLower(strings.TrimSpace(getenv("PROXY_CORE_PRIMARY", "")))
	if primary == "" {
		if mode == "rust" {
			primary = "rust"
		} else {
			primary = "go"
		}
	}
	if primary != "go" && primary != "rust" {
		primary = "go"
	}

	origins := map[string]string{
		"go":   goOrigin,
		"rust": rustOrigin,
	}
	primaryOrigin := origins[primary]
	if primaryOrigin == "" {
		primary = "go"
		primaryOrigin = origins[primary]
	}

	return proxyRuntimeConfig{
		Mode:          mode,
		Primary:       primary,
		PrimaryOrigin: primaryOrigin,
		Origins:       origins,
	}
}

func probeKernelStatus(ctx context.Context, client *http.Client, cfg proxyRuntimeConfig) kernelStatusResponse {
	now := time.Now().UTC()
	status := kernelStatusResponse{
		Mode:      cfg.Mode,
		Primary:   cfg.Primary,
		Kernels:   map[string]kernelHealth{},
		CheckedAt: now,
	}
	for _, name := range []string{"go", "rust"} {
		role := "standby"
		if name == cfg.Primary {
			role = "primary"
		} else if cfg.Mode == "dual" {
			role = "experimental"
		}
		status.Kernels[name] = probeSingleKernel(ctx, client, name, cfg.Origins[name], role, now)
	}
	return status
}

func probeSingleKernel(ctx context.Context, client *http.Client, name, origin, role string, now time.Time) kernelHealth {
	result := kernelHealth{
		Name:       name,
		Origin:     origin,
		Configured: strings.TrimSpace(origin) != "",
		Role:       role,
		CheckedAt:  now,
	}
	if !result.Configured {
		result.Error = "origin not configured"
		return result
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(origin, "/")+"/healthz", nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Healthy = true
		return result
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
		if message, ok := payload["error"].(string); ok && strings.TrimSpace(message) != "" {
			result.Error = message
			return result
		}
	}
	result.Error = resp.Status
	return result
}
