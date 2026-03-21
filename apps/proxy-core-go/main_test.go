package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/service"
	proxyproviders "unifiedsubscriptionproxy/internal/proxy/providers"
)

func TestExecuteAgainstTraceFallsBackToLaterCandidate(t *testing.T) {
	originalNow := timeNowUTC
	timeNowUTC = func() time.Time { return time.Date(2026, 3, 21, 16, 0, 0, 0, time.UTC) }
	defer func() { timeNowUTC = originalNow }()

	data := domain.PlatformData{
		UpstreamAccounts: []domain.UpstreamAccount{
			{ID: "acct-openai-1", Provider: domain.ProviderOpenAI, Status: domain.AccountStatusActive},
			{ID: "acct-codex-1", Provider: domain.ProviderCodex, Status: domain.AccountStatusActive},
		},
	}
	trace := service.DispatchTrace{
		ModelAlias: "gpt-reasoning",
		APIKey:     domain.APIKey{ID: "key-demo", UserID: "user-demo", PackageID: "pkg-hybrid"},
		Candidates: []service.Candidate{
			{AccountID: "acct-openai-1", Provider: domain.ProviderOpenAI, UpstreamModel: "gpt-5", Priority: 20, Weight: 10},
			{AccountID: "acct-codex-1", Provider: domain.ProviderCodex, UpstreamModel: "gpt-5-codex", Priority: 10, Weight: 10},
		},
	}
	stubClient := &stubControlPlaneClient{}
	providers := map[string]proxyproviders.Provider{
		domain.ProviderOpenAI: stubProvider{err: errors.New("openai unavailable")},
		domain.ProviderCodex:  stubProvider{resp: proxyproviders.Response{StatusCode: http.StatusOK, Body: []byte(`{"id":"ok"}`), Headers: http.Header{"Content-Type": []string{"application/json"}}}},
	}
	resp, result, usageLog, err := executeAgainstTrace(context.Background(), stubClient, func(name string) (proxyproviders.Provider, error) {
		provider, ok := providers[name]
		if !ok {
			return nil, errors.New("missing provider")
		}
		return provider, nil
	}, data, trace, []byte(`{"model":"gpt-reasoning"}`), false)
	if err != nil {
		t.Fatalf("executeAgainstTrace returned error: %v", err)
	}
	if result.Provider != domain.ProviderCodex {
		t.Fatalf("expected codex provider, got %s", result.Provider)
	}
	if usageLog.Provider != domain.ProviderCodex || usageLog.Status != "completed" {
		t.Fatalf("unexpected usage log: %#v", usageLog)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected response status: %d", resp.StatusCode)
	}
	if len(stubClient.logs) != 2 {
		t.Fatalf("expected two usage logs, got %d", len(stubClient.logs))
	}
}

func TestExecuteAgainstTraceFallsBackFromAntigravityToGemini(t *testing.T) {
	originalNow := timeNowUTC
	timeNowUTC = func() time.Time { return time.Date(2026, 3, 21, 16, 5, 0, 0, time.UTC) }
	defer func() { timeNowUTC = originalNow }()

	data := domain.PlatformData{
		UpstreamAccounts: []domain.UpstreamAccount{
			{ID: "acct-antigravity-1", Provider: domain.ProviderAntigravity, Status: domain.AccountStatusActive},
			{ID: "acct-gemini-1", Provider: domain.ProviderGemini, Status: domain.AccountStatusActive},
		},
	}
	trace := service.DispatchTrace{
		ModelAlias: "hybrid-premium",
		APIKey:     domain.APIKey{ID: "key-demo", UserID: "user-demo", PackageID: "pkg-hybrid"},
		Candidates: []service.Candidate{
			{AccountID: "acct-antigravity-1", Provider: domain.ProviderAntigravity, UpstreamModel: "hybrid-premium", Priority: 20, Weight: 10},
			{AccountID: "acct-gemini-1", Provider: domain.ProviderGemini, UpstreamModel: "gemini-2.5-pro", Priority: 10, Weight: 10},
		},
	}
	stubClient := &stubControlPlaneClient{}
	providers := map[string]proxyproviders.Provider{
		domain.ProviderAntigravity: stubProvider{resp: proxyproviders.Response{StatusCode: http.StatusServiceUnavailable, Body: []byte(`{"error":{"message":"antigravity down","type":"upstream_error"}}`), Headers: http.Header{"Content-Type": []string{"application/json"}}}},
		domain.ProviderGemini:      stubProvider{resp: proxyproviders.Response{StatusCode: http.StatusOK, Body: []byte(`{"id":"ok"}`), Headers: http.Header{"Content-Type": []string{"application/json"}}}},
	}
	resp, result, usageLog, err := executeAgainstTrace(context.Background(), stubClient, func(name string) (proxyproviders.Provider, error) {
		provider, ok := providers[name]
		if !ok {
			return nil, errors.New("missing provider")
		}
		return provider, nil
	}, data, trace, []byte(`{"model":"hybrid-premium"}`), false)
	if err != nil {
		t.Fatalf("executeAgainstTrace returned error: %v", err)
	}
	if result.Provider != domain.ProviderGemini {
		t.Fatalf("expected gemini provider, got %s", result.Provider)
	}
	if usageLog.Provider != domain.ProviderGemini || usageLog.Status != "completed" {
		t.Fatalf("unexpected usage log: %#v", usageLog)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected response status: %d", resp.StatusCode)
	}
	if len(stubClient.logs) != 2 {
		t.Fatalf("expected two usage logs, got %d", len(stubClient.logs))
	}
	if stubClient.logs[0].Provider != domain.ProviderAntigravity || stubClient.logs[0].ErrorType != "upstream_unavailable" {
		t.Fatalf("expected first log to capture antigravity failure, got %#v", stubClient.logs[0])
	}
}

type stubControlPlaneClient struct {
	logs []domain.UsageLog
}

func (s *stubControlPlaneClient) AppendUsageLog(_ context.Context, log domain.UsageLog) error {
	s.logs = append(s.logs, log)
	return nil
}

type stubProvider struct {
	resp proxyproviders.Response
	err  error
}

func (s stubProvider) Name() string { return "stub" }
func (s stubProvider) Execute(context.Context, proxyproviders.Request) (proxyproviders.Response, error) {
	return s.resp, s.err
}
func (s stubProvider) HealthCheck(context.Context, domain.UpstreamAccount) (proxyproviders.HealthResult, error) {
	return proxyproviders.HealthResult{OK: true, StatusCode: http.StatusOK}, nil
}
