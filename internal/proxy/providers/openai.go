package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type OpenAIProvider struct {
	client *http.Client
}

func NewOpenAIProvider(client *http.Client) *OpenAIProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &OpenAIProvider{client: client}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Execute(ctx context.Context, req Request) (Response, error) {
	apiKey := strings.TrimSpace(req.Account.Meta["api_key"])
	if apiKey == "" {
		return Response{}, fmt.Errorf("openai provider missing api_key for account %s", req.Account.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(req.Account.Meta["base_url"]), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	body, err := withUpstreamModel(req.Body, req.UpstreamModel)
	if err != nil {
		return Response{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}

	headers := make(http.Header)
	for key, values := range resp.Header {
		headers[key] = append([]string(nil), values...)
	}
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

func (p *OpenAIProvider) HealthCheck(ctx context.Context, account domain.UpstreamAccount) (HealthResult, error) {
	apiKey := strings.TrimSpace(account.Meta["api_key"])
	if apiKey == "" {
		return HealthResult{}, fmt.Errorf("openai provider missing api_key for account %s", account.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(account.Meta["base_url"]), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return HealthResult{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
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
		result.Message = "openai account reachable"
	} else {
		result.Message = string(body)
	}
	return result, nil
}

func normalizeOpenAIError(statusCode int, body []byte) []byte {
	if statusCode < 400 {
		return body
	}
	if len(body) == 0 {
		return []byte(`{"error":{"message":"openai upstream error","type":"upstream_error"}}`)
	}
	return body
}
