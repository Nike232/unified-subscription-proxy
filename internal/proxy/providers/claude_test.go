package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

func TestClaudeProviderTransformsRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-claude-key" {
			t.Fatalf("unexpected x-api-key: %s", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Fatalf("unexpected anthropic-version: %s", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["system"] != "follow policy" {
			t.Fatalf("unexpected system prompt: %#v", body["system"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg_123",
			"type":        "message",
			"role":        "assistant",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": "claude ok"},
			},
			"usage": map[string]any{
				"input_tokens":  7,
				"output_tokens": 3,
			},
		})
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.Client())
	resp, err := provider.Execute(context.Background(), Request{
		UpstreamModel: "claude-sonnet-4.5",
		Account: domain.UpstreamAccount{
			ID:       "acct-claude-1",
			Provider: domain.ProviderClaude,
			Meta: map[string]string{
				"api_key":  "test-claude-key",
				"base_url": server.URL,
			},
		},
		Body: []byte(`{"model":"claude-chat","messages":[{"role":"system","content":"follow policy"},{"role":"user","content":"hello"}],"temperature":0.3}`),
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if !strings.Contains(string(resp.Body), "claude ok") {
		t.Fatalf("expected transformed body, got %s", string(resp.Body))
	}
}

func TestClaudeProviderStreamConversion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: content_block_delta\n")
		_, _ = io.WriteString(w, "data: {\"delta\":{\"text\":\"hello\"}}\n\n")
		_, _ = io.WriteString(w, "event: message_stop\n")
		_, _ = io.WriteString(w, "data: {}\n\n")
	}))
	defer server.Close()

	provider := NewClaudeProvider(server.Client())
	resp, err := provider.Execute(context.Background(), Request{
		UpstreamModel: "claude-sonnet-4.5",
		Account: domain.UpstreamAccount{
			ID:       "acct-claude-1",
			Provider: domain.ProviderClaude,
			Meta: map[string]string{
				"api_key":  "test-claude-key",
				"base_url": server.URL,
			},
		},
		Body: []byte(`{"model":"claude-chat","stream":true,"messages":[{"role":"user","content":"hello"}]}`),
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if resp.Stream == nil {
		t.Fatalf("expected stream response")
	}
	defer resp.Stream.Close()
	body, err := io.ReadAll(resp.Stream)
	if err != nil {
		t.Fatalf("failed reading stream: %v", err)
	}
	if !strings.Contains(string(body), "chat.completion.chunk") || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("unexpected stream body: %s", string(body))
	}
}
