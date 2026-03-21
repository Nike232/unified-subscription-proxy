package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type CodexProvider struct {
	client *http.Client
}

func NewCodexProvider(client *http.Client) *CodexProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &CodexProvider{client: client}
}

func (p *CodexProvider) Name() string {
	return domain.ProviderCodex
}

func (p *CodexProvider) Execute(ctx context.Context, req Request) (Response, error) {
	baseURL := strings.TrimRight(firstNonEmpty(req.Account.Meta["base_url"], "https://api.openai.com"), "/")
	body, err := withUpstreamModel(req.Body, req.UpstreamModel)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	applyOpenAICompatibleAuthHeaders(httpReq, req.Account)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	headers := cloneHeaders(resp.Header)
	stream, _ := detectStream(body)
	if stream && resp.StatusCode < 400 {
		return Response{
			StatusCode: resp.StatusCode,
			Headers:    headers,
			Stream:     resp.Body,
		}, nil
	}
	defer resp.Body.Close()
	rawRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       normalizeOpenAIError(resp.StatusCode, rawRespBody),
	}, nil
}

func (p *CodexProvider) HealthCheck(ctx context.Context, account domain.UpstreamAccount) (HealthResult, error) {
	baseURL := strings.TrimRight(firstNonEmpty(account.Meta["base_url"], "https://api.openai.com"), "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return HealthResult{}, err
	}
	applyOpenAICompatibleAuthHeaders(httpReq, account)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return HealthResult{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	result := HealthResult{
		OK:         resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
	}
	if result.OK {
		result.Message = "codex account reachable"
	} else {
		result.Message = string(body)
	}
	return result, nil
}

func applyOpenAICompatibleAuthHeaders(req *http.Request, account domain.UpstreamAccount) {
	token := strings.TrimSpace(firstNonEmpty(account.Meta["access_token"], account.Meta["api_key"]))
	if token == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if org := strings.TrimSpace(account.Meta["organization_id"]); org != "" {
		req.Header.Set("OpenAI-Organization", org)
	}
}

func cloneHeaders(src http.Header) http.Header {
	out := make(http.Header)
	for key, values := range src {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func writeSSEChunk(w io.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", body)
	return err
}

func stringValue(value any) string {
	if out, ok := value.(string); ok {
		return strings.TrimSpace(out)
	}
	return ""
}

func boolValue(value any) bool {
	if out, ok := value.(bool); ok {
		return out
	}
	return false
}

func intValue(value any, fallback int) int {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case int:
		if typed > 0 {
			return typed
		}
	}
	return fallback
}

func withUpstreamModel(raw []byte, upstreamModel string) ([]byte, error) {
	if strings.TrimSpace(upstreamModel) == "" {
		return raw, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	payload["model"] = upstreamModel
	return json.Marshal(payload)
}
