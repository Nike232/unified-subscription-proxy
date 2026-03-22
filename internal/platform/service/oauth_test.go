package service

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/store"
)

func TestBuildOAuthAuthorizeURL(t *testing.T) {
	u, err := BuildOAuthAuthorizeURL(OAuthProviderConfig{
		Provider:     domain.ProviderClaude,
		ClientID:     "client-id",
		AuthorizeURL: "https://example.com/oauth/authorize",
		RedirectURL:  "http://127.0.0.1/callback",
		Scopes:       []string{"read", "write"},
	}, domain.OAuthSession{State: "state-123"})
	if err != nil {
		t.Fatalf("BuildOAuthAuthorizeURL returned error: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed parsing url: %v", err)
	}
	if parsed.Query().Get("state") != "state-123" {
		t.Fatalf("unexpected state: %s", parsed.Query().Get("state"))
	}
	if parsed.Query().Get("client_id") != "client-id" {
		t.Fatalf("unexpected client id: %s", parsed.Query().Get("client_id"))
	}
}

func TestBuildOAuthAuthorizeURLWithPKCE(t *testing.T) {
	u, err := BuildOAuthAuthorizeURL(OAuthProviderConfig{
		Provider:              domain.ProviderGemini,
		ClientID:              "client-id",
		AuthorizeURL:          "https://example.com/oauth/authorize",
		RedirectURL:           "http://127.0.0.1/callback",
		Scopes:                []string{"scope-a"},
		UsePKCE:               true,
		AccessType:            "offline",
		Prompt:                "consent",
		IncludeGrantedScopes:  true,
		ExtraAuthorizeParams:  map[string]string{"foo": "bar"},
	}, domain.OAuthSession{State: "state-456", CodeVerifier: "verifier-123"})
	if err != nil {
		t.Fatalf("BuildOAuthAuthorizeURL returned error: %v", err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("failed parsing url: %v", err)
	}
	q := parsed.Query()
	if q.Get("state") != "state-456" {
		t.Fatalf("unexpected state: %s", q.Get("state"))
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("expected PKCE params, got %s", parsed.RawQuery)
	}
	if q.Get("access_type") != "offline" || q.Get("prompt") != "consent" {
		t.Fatalf("expected offline prompt params, got %s", parsed.RawQuery)
	}
	if q.Get("include_granted_scopes") != "true" || q.Get("foo") != "bar" {
		t.Fatalf("expected extra params, got %s", parsed.RawQuery)
	}
}

func TestCreateAndCompleteOAuthSession(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))
	session, providerName, err := svc.CreateOAuthSession("acct-claude-1", "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("CreateOAuthSession returned error: %v", err)
	}
	if providerName != domain.ProviderClaude {
		t.Fatalf("unexpected provider name: %s", providerName)
	}
	if strings.TrimSpace(session.CodeVerifier) == "" {
		t.Fatalf("expected code verifier to be generated")
	}
	account, completed, err := svc.CompleteOAuthSession(session.State, TokenPayload{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CompleteOAuthSession returned error: %v", err)
	}
	if account.Meta["access_token"] != "access" || account.Status != domain.AccountStatusActive {
		t.Fatalf("unexpected account after callback: %#v", account)
	}
	if completed.Status != "completed" {
		t.Fatalf("unexpected session status: %#v", completed)
	}
}

func TestMarkAccountInvalid(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))
	account, err := svc.MarkAccountInvalid("acct-claude-1", "auth failed")
	if err != nil {
		t.Fatalf("MarkAccountInvalid returned error: %v", err)
	}
	if account.Status != domain.AccountStatusInvalid {
		t.Fatalf("expected invalid status, got %#v", account)
	}
	if !strings.Contains(account.Meta["last_refresh_error"], "auth failed") {
		t.Fatalf("expected refresh error to be stored")
	}
}

func TestRefreshAccountTokensPersistsToFileStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "platform.json")
	svc := New(store.NewFileStore(path))

	updated, err := svc.RefreshAccountTokens("acct-claude-1", TokenPayload{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().UTC().Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("RefreshAccountTokens returned error: %v", err)
	}
	if updated.Meta["access_token"] != "new-access" {
		t.Fatalf("unexpected updated account: %#v", updated)
	}

	data, err := svc.Data()
	if err != nil {
		t.Fatalf("Data returned error: %v", err)
	}
	account, err := findAccountByID(data.UpstreamAccounts, "acct-claude-1")
	if err != nil {
		t.Fatalf("findAccountByID returned error: %v", err)
	}
	if account.Meta["access_token"] != "new-access" || account.Meta["refresh_token"] != "new-refresh" {
		t.Fatalf("expected refreshed tokens to persist, got %#v", account.Meta)
	}
}

func TestRecordUsageOutcomeEntersCooldownAndRecovers(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))
	policy := AccountHealthPolicy{FailureThreshold: 2, Cooldown: time.Minute}

	updated, err := svc.RecordUsageOutcome("acct-openai-1", "provider_error", "boom", policy)
	if err != nil {
		t.Fatalf("RecordUsageOutcome first call returned error: %v", err)
	}
	if AccountConsecutiveFailures(updated) != 1 {
		t.Fatalf("expected 1 failure, got %d", AccountConsecutiveFailures(updated))
	}

	updated, err = svc.RecordUsageOutcome("acct-openai-1", "provider_error", "boom again", policy)
	if err != nil {
		t.Fatalf("RecordUsageOutcome second call returned error: %v", err)
	}
	if _, ok := AccountCooldownUntil(updated); !ok {
		t.Fatalf("expected cooldown to be set")
	}

	updated, err = svc.RecordUsageOutcome("acct-openai-1", "", "", policy)
	if err != nil {
		t.Fatalf("RecordUsageOutcome recovery returned error: %v", err)
	}
	if AccountConsecutiveFailures(updated) != 0 {
		t.Fatalf("expected failures to reset, got %d", AccountConsecutiveFailures(updated))
	}
	if _, ok := AccountCooldownUntil(updated); ok {
		t.Fatalf("expected cooldown to be cleared")
	}
}

func TestCleanupOAuthSessionsRemovesCompletedAndExpired(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))

	data, err := svc.Data()
	if err != nil {
		t.Fatalf("Data returned error: %v", err)
	}
	data.OAuthSessions = []domain.OAuthSession{
		{ID: "keep", State: "keep", Status: "pending", ExpiresAt: time.Now().UTC().Add(5 * time.Minute)},
		{ID: "done", State: "done", Status: "completed", ExpiresAt: time.Now().UTC().Add(5 * time.Minute)},
		{ID: "old", State: "old", Status: "pending", ExpiresAt: time.Now().UTC().Add(-time.Minute)},
	}
	if err := svc.store.Save(data); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	removed, err := svc.CleanupOAuthSessions(time.Now().UTC())
	if err != nil {
		t.Fatalf("CleanupOAuthSessions returned error: %v", err)
	}
	if removed != 2 {
		t.Fatalf("expected 2 sessions removed, got %d", removed)
	}
	updatedData, err := svc.Data()
	if err != nil {
		t.Fatalf("Data returned error: %v", err)
	}
	if len(updatedData.OAuthSessions) != 1 || updatedData.OAuthSessions[0].ID != "keep" {
		t.Fatalf("unexpected sessions after cleanup: %#v", updatedData.OAuthSessions)
	}
}

func TestRecordUsageOutcomeMarksAuthFailedInvalid(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))

	updated, err := svc.RecordUsageOutcome("acct-claude-1", "auth_failed", "token expired", AccountHealthPolicy{})
	if err != nil {
		t.Fatalf("RecordUsageOutcome returned error: %v", err)
	}
	if updated.Status != domain.AccountStatusInvalid {
		t.Fatalf("expected invalid status, got %#v", updated)
	}
	if !strings.Contains(updated.Meta["last_failure_reason"], "token expired") {
		t.Fatalf("expected last_failure_reason to be stored")
	}
}
