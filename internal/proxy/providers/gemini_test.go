package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

func TestGeminiProviderTransformsRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "key=test-gemini-key") {
			t.Fatalf("expected gemini api key query param")
		}
		if !strings.HasPrefix(r.URL.Path, "/v1beta/models/gemini-2.5-pro:generateContent") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]any{{"text": "gemini says hi"}},
					},
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     5,
				"candidatesTokenCount": 3,
				"totalTokenCount":      8,
			},
		})
	}))
	defer server.Close()

	provider := NewGeminiProvider(server.Client())
	resp, err := provider.Execute(context.Background(), Request{
		UpstreamModel: "gemini-2.5-pro",
		Account: domain.UpstreamAccount{
			ID:       "acct-gemini-1",
			Provider: domain.ProviderGemini,
			Meta: map[string]string{
				"api_key":  "test-gemini-key",
				"base_url": server.URL,
			},
		},
		Body: []byte(`{"model":"gemini-pro","messages":[{"role":"user","content":"hello"}]}`),
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if !strings.Contains(string(resp.Body), "gemini says hi") {
		t.Fatalf("expected transformed body, got %s", string(resp.Body))
	}
}
