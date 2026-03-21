package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

func TestOpenAIProviderForwardsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-openai-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "cmpl", "object": "chat.completion"})
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.Client())
	resp, err := provider.Execute(context.Background(), Request{
		Account: domain.UpstreamAccount{
			ID:       "acct-openai-1",
			Provider: domain.ProviderOpenAI,
			Meta: map[string]string{
				"api_key":  "test-openai-key",
				"base_url": server.URL,
			},
		},
		Body: []byte(`{"model":"gpt-fast","messages":[{"role":"user","content":"hello"}]}`),
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
