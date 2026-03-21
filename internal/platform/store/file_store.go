package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Load() (domain.PlatformData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked()
}

func (s *FileStore) Save(data domain.PlatformData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(data)
}

func (s *FileStore) Mutate(fn func(*domain.PlatformData) error) (domain.PlatformData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadUnlocked()
	if err != nil {
		return domain.PlatformData{}, err
	}
	if err := fn(&data); err != nil {
		return domain.PlatformData{}, err
	}
	if err := s.saveUnlocked(data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (s *FileStore) loadUnlocked() (domain.PlatformData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			data := bootstrapData()
			if err := s.saveUnlocked(data); err != nil {
				return domain.PlatformData{}, err
			}
			return data, nil
		}
		return domain.PlatformData{}, err
	}

	var data domain.PlatformData
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (s *FileStore) saveUnlocked(data domain.PlatformData) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func bootstrapData() domain.PlatformData {
	now := time.Now().UTC()
	return domain.PlatformData{
		Users: []domain.User{
			{ID: "user-admin", Email: "admin@example.com", Name: "Admin", Role: "admin"},
			{ID: "user-demo", Email: "demo@example.com", Name: "Demo User", Role: "user"},
		},
		UpstreamAccounts: []domain.UpstreamAccount{
			{ID: "acct-claude-1", Provider: domain.ProviderClaude, DisplayName: "Claude Shared 01", Email: "claude-01@example.com", AuthMode: "oauth", Status: domain.AccountStatusActive, Tier: "pro", Weight: 10, Priority: 10, SupportsModels: []string{"claude-sonnet-4.5", "claude-opus-4.1"}, LastRefreshedAt: now},
			{ID: "acct-codex-1", Provider: domain.ProviderCodex, DisplayName: "Codex Shared 01", Email: "codex-01@example.com", AuthMode: "oauth", Status: domain.AccountStatusActive, Tier: "pro", Weight: 10, Priority: 10, SupportsModels: []string{"gpt-5-codex", "gpt-5"}, LastRefreshedAt: now},
			{ID: "acct-openai-1", Provider: domain.ProviderOpenAI, DisplayName: "OpenAI Shared 01", Email: "openai-01@example.com", AuthMode: "api_key", Status: domain.AccountStatusActive, Tier: "enterprise", Weight: 10, Priority: 20, SupportsModels: []string{"gpt-5", "gpt-4.1"}, LastRefreshedAt: now},
			{ID: "acct-gemini-1", Provider: domain.ProviderGemini, DisplayName: "Gemini Shared 01", Email: "gemini-01@example.com", AuthMode: "oauth", Status: domain.AccountStatusActive, Tier: "ultra", Weight: 10, Priority: 10, SupportsModels: []string{"gemini-2.5-pro", "gemini-2.5-flash"}, LastRefreshedAt: now},
			{ID: "acct-antigravity-1", Provider: domain.ProviderAntigravity, DisplayName: "Antigravity Shared 01", Email: "antigravity-01@example.com", AuthMode: "oauth", Status: domain.AccountStatusActive, Tier: "ultra", Weight: 10, Priority: 10, SupportsModels: []string{"claude-sonnet-4.5", "gemini-2.5-pro"}, LastRefreshedAt: now},
		},
		ServicePackages: []domain.ServicePackage{
			{
				ID:          "pkg-basic",
				Name:        "Basic",
				Tier:        "basic",
				Description: "GPT + Gemini starter package",
				ProviderAccess: []domain.ProviderAccess{
					{Provider: domain.ProviderOpenAI, Models: []string{"gpt-fast", "gpt-reasoning"}},
					{Provider: domain.ProviderGemini, Models: []string{"gemini-fast", "gemini-pro"}},
				},
				AllowCrossProviderFallback: false,
				DefaultConcurrency:         2,
			},
			{
				ID:          "pkg-advanced",
				Name:        "Advanced",
				Tier:        "advanced",
				Description: "Claude + Codex + Antigravity",
				ProviderAccess: []domain.ProviderAccess{
					{Provider: domain.ProviderClaude, Models: []string{"claude-chat"}},
					{Provider: domain.ProviderCodex, Models: []string{"gpt-reasoning"}},
					{Provider: domain.ProviderAntigravity, Models: []string{"hybrid-premium"}},
				},
				AllowCrossProviderFallback: true,
				DefaultConcurrency:         5,
			},
			{
				ID:          "pkg-hybrid",
				Name:        "Hybrid",
				Tier:        "hybrid",
				Description: "All providers with unified aliases",
				ProviderAccess: []domain.ProviderAccess{
					{Provider: domain.ProviderOpenAI, Models: []string{"gpt-fast", "gpt-reasoning"}},
					{Provider: domain.ProviderGemini, Models: []string{"gemini-fast", "gemini-pro"}},
					{Provider: domain.ProviderClaude, Models: []string{"claude-chat"}},
					{Provider: domain.ProviderCodex, Models: []string{"gpt-reasoning"}},
					{Provider: domain.ProviderAntigravity, Models: []string{"hybrid-premium"}},
				},
				AllowCrossProviderFallback: true,
				DefaultConcurrency:         10,
			},
		},
		ModelAliasPolicies: []domain.ModelAliasPolicy{
			{Alias: "gpt-fast", Targets: []domain.ModelTarget{{Provider: domain.ProviderOpenAI, UpstreamModel: "gpt-4.1", Priority: 10}}},
			{Alias: "gpt-reasoning", Targets: []domain.ModelTarget{{Provider: domain.ProviderOpenAI, UpstreamModel: "gpt-5", Priority: 20}, {Provider: domain.ProviderCodex, UpstreamModel: "gpt-5-codex", Priority: 10}}},
			{Alias: "gemini-fast", Targets: []domain.ModelTarget{{Provider: domain.ProviderGemini, UpstreamModel: "gemini-2.5-flash", Priority: 10}}},
			{Alias: "gemini-pro", Targets: []domain.ModelTarget{{Provider: domain.ProviderGemini, UpstreamModel: "gemini-2.5-pro", Priority: 10}}},
			{Alias: "claude-chat", Targets: []domain.ModelTarget{{Provider: domain.ProviderClaude, UpstreamModel: "claude-sonnet-4.5", Priority: 10}}},
			{Alias: "hybrid-premium", Targets: []domain.ModelTarget{{Provider: domain.ProviderAntigravity, UpstreamModel: "claude-sonnet-4.5", Priority: 20}, {Provider: domain.ProviderGemini, UpstreamModel: "gemini-2.5-pro", Priority: 10}}},
		},
		AccountPoolPolicy: []domain.AccountPoolPolicy{
			{ID: "pool-gpt-fast", Alias: "gpt-fast", Provider: domain.ProviderOpenAI, AccountIDs: []string{"acct-openai-1"}, Strategy: "priority"},
			{ID: "pool-gpt-reasoning", Alias: "gpt-reasoning", Provider: domain.ProviderOpenAI, AccountIDs: []string{"acct-openai-1", "acct-codex-1"}, Strategy: "fallback"},
			{ID: "pool-gemini-pro", Alias: "gemini-pro", Provider: domain.ProviderGemini, AccountIDs: []string{"acct-gemini-1"}, Strategy: "priority"},
			{ID: "pool-claude-chat", Alias: "claude-chat", Provider: domain.ProviderClaude, AccountIDs: []string{"acct-claude-1"}, Strategy: "priority"},
			{ID: "pool-hybrid-premium", Alias: "hybrid-premium", Provider: domain.ProviderAntigravity, AccountIDs: []string{"acct-antigravity-1", "acct-gemini-1"}, Strategy: "fallback"},
		},
		Subscriptions: []domain.Subscription{
			{ID: "sub-demo", UserID: "user-demo", PackageID: "pkg-hybrid", Status: domain.SubscriptionStatusActive, StartsAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour), AssignedBy: "user-admin", Description: "demo bootstrap subscription"},
		},
		APIKeys: []domain.APIKey{
			{ID: "key-demo", Key: "usp_demo_key", UserID: "user-demo", PackageID: "pkg-hybrid", Status: "active", CreatedAt: now},
		},
	}
}
