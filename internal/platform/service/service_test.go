package service

import (
	"path/filepath"
	"testing"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/store"
)

func TestResolveDispatchUsesAllowedPackageRoute(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))

	result, err := svc.ResolveDispatch("gpt-reasoning", "usp_demo_key")
	if err != nil {
		t.Fatalf("ResolveDispatch returned error: %v", err)
	}
	if result.Provider == "" {
		t.Fatalf("expected provider to be selected")
	}
	if result.UpstreamModel == "" {
		t.Fatalf("expected upstream model to be selected")
	}
}

func TestResolveDispatchInDataRejectsPackageWithoutAccess(t *testing.T) {
	data := store.BootstrapData()
	data.APIKeys = []domain.APIKey{
		{ID: "key-basic", Key: "usp_basic_key", UserID: "user-demo", PackageID: "pkg-basic", Status: "active"},
	}

	_, _, err := ResolveDispatchInData(data, "claude-chat", "usp_basic_key")
	if err == nil {
		t.Fatalf("expected package access error")
	}
}

func TestExplainDispatchSkipsExpiredAccountWithoutRefresh(t *testing.T) {
	data := store.BootstrapData()
	for i := range data.UpstreamAccounts {
		if data.UpstreamAccounts[i].Provider == domain.ProviderClaude {
			data.UpstreamAccounts[i].Meta = map[string]string{
				"access_token": "expired",
				"expires_at":   time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
			}
		}
	}
	_, err := ExplainDispatchInData(data, "claude-chat", "usp_demo_key")
	if err == nil {
		t.Fatalf("expected expired account to be rejected")
	}
}

func TestExplainDispatchIncludesRefreshableExpiredCandidate(t *testing.T) {
	data := store.BootstrapData()
	for i := range data.UpstreamAccounts {
		if data.UpstreamAccounts[i].Provider == domain.ProviderClaude {
			data.UpstreamAccounts[i].Meta = map[string]string{
				"access_token":  "expired",
				"refresh_token": "refresh",
				"expires_at":    time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
			}
		}
	}
	trace, err := ExplainDispatchInData(data, "claude-chat", "usp_demo_key")
	if err != nil {
		t.Fatalf("expected refreshable candidate to remain dispatchable: %v", err)
	}
	if len(trace.Candidates) == 0 || !trace.Candidates[0].CanRefresh {
		t.Fatalf("expected candidate to be refreshable")
	}
}

func TestExplainDispatchSkipsCooldownCandidate(t *testing.T) {
	data := store.BootstrapData()
	now := time.Now().UTC()
	for i := range data.UpstreamAccounts {
		if data.UpstreamAccounts[i].Provider == domain.ProviderOpenAI {
			if data.UpstreamAccounts[i].Meta == nil {
				data.UpstreamAccounts[i].Meta = map[string]string{}
			}
			data.UpstreamAccounts[i].Meta["cooldown_until"] = now.Add(5 * time.Minute).Format(time.RFC3339)
		}
		if data.UpstreamAccounts[i].Provider == domain.ProviderCodex {
			if data.UpstreamAccounts[i].Meta == nil {
				data.UpstreamAccounts[i].Meta = map[string]string{}
			}
			delete(data.UpstreamAccounts[i].Meta, "cooldown_until")
		}
	}

	trace, err := ExplainDispatchInData(data, "gpt-reasoning", "usp_demo_key")
	if err != nil {
		t.Fatalf("expected fallback candidate to be selected: %v", err)
	}
	if trace.Selected.Provider != domain.ProviderCodex {
		t.Fatalf("expected codex to be selected after openai cooldown, got %s", trace.Selected.Provider)
	}
	if len(trace.Candidates) == 0 || trace.Candidates[0].SkipReason != "cooldown" {
		t.Fatalf("expected first candidate to be skipped by cooldown: %#v", trace.Candidates)
	}
}
