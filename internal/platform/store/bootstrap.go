package store

import (
	"os"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

func BootstrapData() domain.PlatformData {
	now := time.Now().UTC()
	return domain.PlatformData{
		Users: []domain.User{
			{ID: "user-admin", Email: "admin@example.com", Name: "Admin", Role: "admin", PasswordHash: envOrDefault("ADMIN_BOOTSTRAP_PASSWORD", "admin123")},
			{ID: "user-demo", Email: "demo@example.com", Name: "Demo User", Role: "user", PasswordHash: envOrDefault("USER_BOOTSTRAP_PASSWORD", "demo123")},
		},
		UpstreamAccounts: []domain.UpstreamAccount{
			{
				ID:             "acct-claude-1",
				Provider:       domain.ProviderClaude,
				DisplayName:    "Claude Shared 01",
				Email:          "claude-01@example.com",
				AuthMode:       "oauth",
				Status:         domain.AccountStatusActive,
				Tier:           "pro",
				Weight:         10,
				Priority:       10,
				SupportsModels: []string{"claude-sonnet-4.5", "claude-opus-4.1"},
				Meta: map[string]string{
					"api_key":      os.Getenv("CLAUDE_API_KEY"),
					"access_token": os.Getenv("CLAUDE_ACCESS_TOKEN"),
					"base_url":     envOrDefault("CLAUDE_BASE_URL", "https://api.anthropic.com"),
				},
				LastRefreshedAt: now,
			},
			{
				ID:             "acct-codex-1",
				Provider:       domain.ProviderCodex,
				DisplayName:    "Codex Shared 01",
				Email:          "codex-01@example.com",
				AuthMode:       "oauth",
				Status:         domain.AccountStatusActive,
				Tier:           "pro",
				Weight:         10,
				Priority:       10,
				SupportsModels: []string{"gpt-5-codex", "gpt-5"},
				Meta: map[string]string{
					"api_key":         os.Getenv("CODEX_API_KEY"),
					"access_token":    os.Getenv("CODEX_ACCESS_TOKEN"),
					"organization_id": os.Getenv("CODEX_ORGANIZATION_ID"),
					"base_url":        envOrDefault("CODEX_BASE_URL", "https://api.openai.com"),
				},
				LastRefreshedAt: now,
			},
			{
				ID:             "acct-openai-1",
				Provider:       domain.ProviderOpenAI,
				DisplayName:    "OpenAI Shared 01",
				Email:          "openai-01@example.com",
				AuthMode:       "api_key",
				Status:         domain.AccountStatusActive,
				Tier:           "enterprise",
				Weight:         10,
				Priority:       20,
				SupportsModels: []string{"gpt-5", "gpt-4.1"},
				Meta: map[string]string{
					"api_key":  os.Getenv("OPENAI_API_KEY"),
					"base_url": envOrDefault("OPENAI_BASE_URL", "https://api.openai.com"),
				},
				LastRefreshedAt: now,
			},
			{
				ID:             "acct-gemini-1",
				Provider:       domain.ProviderGemini,
				DisplayName:    "Gemini Shared 01",
				Email:          "gemini-01@example.com",
				AuthMode:       "api_key",
				Status:         domain.AccountStatusActive,
				Tier:           "ultra",
				Weight:         10,
				Priority:       10,
				SupportsModels: []string{"gemini-2.5-pro", "gemini-2.5-flash"},
				Meta: map[string]string{
					"api_key":  os.Getenv("GEMINI_API_KEY"),
					"base_url": envOrDefault("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com"),
				},
				LastRefreshedAt: now,
			},
			{
				ID:             "acct-antigravity-1",
				Provider:       domain.ProviderAntigravity,
				DisplayName:    "Antigravity Shared 01",
				Email:          "antigravity-01@example.com",
				AuthMode:       "oauth",
				Status:         domain.AccountStatusActive,
				Tier:           "ultra",
				Weight:         10,
				Priority:       15,
				SupportsModels: []string{"hybrid-premium"},
				Meta: map[string]string{
					"api_key":      os.Getenv("ANTIGRAVITY_API_KEY"),
					"access_token": os.Getenv("ANTIGRAVITY_ACCESS_TOKEN"),
					"tenant_id":    os.Getenv("ANTIGRAVITY_TENANT_ID"),
					"base_url":     envOrDefault("ANTIGRAVITY_BASE_URL", "https://api.antigravity.example"),
				},
				LastRefreshedAt: now,
			},
		},
		ServicePackages: []domain.ServicePackage{
			{
				ID:           "pkg-basic",
				Name:         "Basic",
				DisplayName:  "Basic Monthly",
				Tier:         "basic",
				Description:  "GPT + Gemini starter package",
				PriceCents:   1900,
				BillingCycle: "monthly",
				IsActive:     true,
				ProviderAccess: []domain.ProviderAccess{
					{Provider: domain.ProviderOpenAI, Models: []string{"gpt-fast", "gpt-reasoning"}},
					{Provider: domain.ProviderGemini, Models: []string{"gemini-fast", "gemini-pro"}},
				},
				AllowCrossProviderFallback: false,
				DefaultConcurrency:         2,
			},
			{
				ID:           "pkg-advanced",
				Name:         "Advanced",
				DisplayName:  "Advanced Monthly",
				Tier:         "advanced",
				Description:  "Claude + Codex + Antigravity",
				PriceCents:   4900,
				BillingCycle: "monthly",
				IsActive:     true,
				ProviderAccess: []domain.ProviderAccess{
					{Provider: domain.ProviderClaude, Models: []string{"claude-chat"}},
					{Provider: domain.ProviderCodex, Models: []string{"gpt-reasoning"}},
					{Provider: domain.ProviderAntigravity, Models: []string{"hybrid-premium"}},
					{Provider: domain.ProviderGemini, Models: []string{"hybrid-premium"}},
				},
				AllowCrossProviderFallback: true,
				DefaultConcurrency:         5,
			},
			{
				ID:           "pkg-hybrid",
				Name:         "Hybrid",
				DisplayName:  "Hybrid Monthly",
				Tier:         "hybrid",
				Description:  "All providers with unified aliases",
				PriceCents:   7900,
				BillingCycle: "monthly",
				IsActive:     true,
				ProviderAccess: []domain.ProviderAccess{
					{Provider: domain.ProviderOpenAI, Models: []string{"gpt-fast", "gpt-reasoning"}},
					{Provider: domain.ProviderGemini, Models: []string{"gemini-fast", "gemini-pro", "hybrid-premium"}},
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
			{Alias: "hybrid-premium", Targets: []domain.ModelTarget{{Provider: domain.ProviderAntigravity, UpstreamModel: "hybrid-premium", Priority: 20}, {Provider: domain.ProviderGemini, UpstreamModel: "gemini-2.5-pro", Priority: 10}}},
		},
		AccountPoolPolicy: []domain.AccountPoolPolicy{
			{ID: "pool-gpt-fast", Alias: "gpt-fast", Provider: domain.ProviderOpenAI, AccountIDs: []string{"acct-openai-1"}, Strategy: "priority"},
			{ID: "pool-gpt-reasoning", Alias: "gpt-reasoning", Provider: domain.ProviderOpenAI, AccountIDs: []string{"acct-openai-1", "acct-codex-1"}, Strategy: "fallback"},
			{ID: "pool-gemini-pro", Alias: "gemini-pro", Provider: domain.ProviderGemini, AccountIDs: []string{"acct-gemini-1"}, Strategy: "priority"},
			{ID: "pool-claude-chat", Alias: "claude-chat", Provider: domain.ProviderClaude, AccountIDs: []string{"acct-claude-1"}, Strategy: "priority"},
			{ID: "pool-hybrid-premium", Alias: "hybrid-premium", Provider: domain.ProviderAntigravity, AccountIDs: []string{"acct-antigravity-1", "acct-gemini-1"}, Strategy: "fallback"},
		},
		Subscriptions: []domain.Subscription{
			{ID: "sub-demo", UserID: "user-demo", PackageID: "pkg-hybrid", Status: domain.SubscriptionStatusActive, StartsAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour), AssignedBy: "user-admin", Description: "demo bootstrap subscription", AutoRenew: false},
		},
		APIKeys: []domain.APIKey{
			{ID: "key-demo", Key: "usp_demo_key", UserID: "user-demo", PackageID: "pkg-hybrid", Status: "active", CreatedAt: now},
		},
		AuthSessions:  []domain.AuthSession{},
		Orders:        []domain.Order{},
		Payments:      []domain.Payment{},
		WebhookEvents: []domain.WebhookEvent{},
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
