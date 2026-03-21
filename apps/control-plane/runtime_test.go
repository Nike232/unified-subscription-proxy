package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeKernelStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := proxyRuntimeConfig{
		Mode:          "dual",
		Primary:       "go",
		PrimaryOrigin: server.URL,
		Origins: map[string]string{
			"go":   server.URL,
			"rust": "http://127.0.0.1:65534",
		},
	}
	status := probeKernelStatus(context.Background(), http.DefaultClient, cfg)
	if !status.Kernels["go"].Healthy {
		t.Fatalf("expected go kernel healthy: %#v", status.Kernels["go"])
	}
	if status.Kernels["rust"].Healthy {
		t.Fatalf("expected rust kernel unhealthy for invalid address")
	}
	if status.Kernels["go"].Role != "primary" {
		t.Fatalf("expected go kernel to be primary")
	}
}

func TestLoadProxyRuntimeConfigDefaults(t *testing.T) {
	t.Setenv("PROXY_CORE_MODE", "")
	t.Setenv("PROXY_CORE_PRIMARY", "")
	t.Setenv("PROXY_CORE_ORIGIN", "http://127.0.0.1:8081")
	t.Setenv("PROXY_CORE_GO_ORIGIN", "")
	t.Setenv("PROXY_CORE_RUST_ORIGIN", "")

	cfg := loadProxyRuntimeConfig()
	if cfg.Mode != "go" {
		t.Fatalf("expected default mode go, got %s", cfg.Mode)
	}
	if cfg.Primary != "go" {
		t.Fatalf("expected default primary go, got %s", cfg.Primary)
	}
	if cfg.PrimaryOrigin != "http://127.0.0.1:8081" {
		t.Fatalf("unexpected primary origin: %s", cfg.PrimaryOrigin)
	}
}
