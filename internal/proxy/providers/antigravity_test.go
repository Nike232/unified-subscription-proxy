package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

func TestAntigravityProviderUsesCompatibleChatEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ag-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "ag-token" {
			t.Fatalf("unexpected x-api-key header: %s", got)
		}
		if got := r.Header.Get("X-Tenant-ID"); got != "tenant-1" {
			t.Fatalf("unexpected tenant header: %s", got)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["model"] != "hybrid-premium" {
			t.Fatalf("expected upstream model override, got %#v", body["model"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "cmpl-antigravity",
			"object": "chat.completion",
			"model":  body["model"],
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "antigravity ok",
				},
				"finish_reason": "stop",
			}},
		})
	}))
	defer server.Close()

	provider := NewAntigravityProvider(server.Client())
	resp, err := provider.Execute(context.Background(), Request{
		UpstreamModel: "hybrid-premium",
		Account: domain.UpstreamAccount{
			ID:       "acct-antigravity-1",
			Provider: domain.ProviderAntigravity,
			Meta: map[string]string{
				"access_token": "ag-token",
				"tenant_id":    "tenant-1",
				"base_url":     server.URL,
			},
		},
		Body: []byte(`{"model":"hybrid-premium","messages":[{"role":"user","content":"hello"}]}`),
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestAntigravityHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"id":"hybrid-premium"}]}`))
	}))
	defer server.Close()

	provider := NewAntigravityProvider(server.Client())
	result, err := provider.HealthCheck(context.Background(), domain.UpstreamAccount{
		ID:       "acct-antigravity-1",
		Provider: domain.ProviderAntigravity,
		Meta: map[string]string{
			"api_key":  "ag-token",
			"base_url": server.URL,
		},
	})
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if !result.OK || result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected health result: %#v", result)
	}
}
