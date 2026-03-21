package main

import (
	"context"
	"net"
	"net/http"
	"testing"
)

func TestProbeKernelStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping kernel probe test because local listen is unavailable: %v", err)
	}
	server := &http.Server{Handler: handler}
	defer server.Close()
	go func() {
		_ = server.Serve(listener)
	}()

	cfg := proxyRuntimeConfig{
		Mode:          "dual",
		Primary:       "go",
		PrimaryOrigin: "http://" + listener.Addr().String(),
		Origins: map[string]string{
			"go":   "http://" + listener.Addr().String(),
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
