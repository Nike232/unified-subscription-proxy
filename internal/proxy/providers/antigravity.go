package providers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type AntigravityProvider struct {
	client *http.Client
}

func NewAntigravityProvider(client *http.Client) *AntigravityProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &AntigravityProvider{client: client}
}

func (p *AntigravityProvider) Name() string {
	return domain.ProviderAntigravity
}

func (p *AntigravityProvider) Execute(ctx context.Context, req Request) (Response, error) {
	baseURL := strings.TrimRight(firstNonEmpty(req.Account.Meta["base_url"], "https://api.antigravity.example"), "/")
	body, err := withUpstreamModel(req.Body, req.UpstreamModel)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	applyAntigravityAuthHeaders(httpReq, req.Account)
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

func (p *AntigravityProvider) HealthCheck(ctx context.Context, account domain.UpstreamAccount) (HealthResult, error) {
	baseURL := strings.TrimRight(firstNonEmpty(account.Meta["base_url"], "https://api.antigravity.example"), "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return HealthResult{}, err
	}
	applyAntigravityAuthHeaders(httpReq, account)
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
		result.Message = "antigravity account reachable"
	} else {
		result.Message = string(body)
	}
	return result, nil
}

func applyAntigravityAuthHeaders(req *http.Request, account domain.UpstreamAccount) {
	token := strings.TrimSpace(firstNonEmpty(account.Meta["access_token"], account.Meta["api_key"]))
	if token == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-API-Key", token)
	if tenant := strings.TrimSpace(account.Meta["tenant_id"]); tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
}
