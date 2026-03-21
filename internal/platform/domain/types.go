package domain

import "time"

const (
	ProviderClaude      = "claude"
	ProviderCodex       = "codex"
	ProviderOpenAI      = "openai"
	ProviderGemini      = "gemini"
	ProviderAntigravity = "antigravity"
)

const (
	AccountStatusActive   = "active"
	AccountStatusDisabled = "disabled"
	AccountStatusInvalid  = "invalid"
)

const (
	SubscriptionStatusActive  = "active"
	SubscriptionStatusExpired = "expired"
)

type PlatformData struct {
	Users              []User              `json:"users"`
	UpstreamAccounts   []UpstreamAccount   `json:"upstream_accounts"`
	ServicePackages    []ServicePackage    `json:"service_packages"`
	ModelAliasPolicies []ModelAliasPolicy  `json:"model_alias_policies"`
	AccountPoolPolicy  []AccountPoolPolicy `json:"account_pool_policy"`
	Subscriptions      []Subscription      `json:"subscriptions"`
	APIKeys            []APIKey            `json:"api_keys"`
	UsageLogs          []UsageLog          `json:"usage_logs"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type UpstreamAccount struct {
	ID              string            `json:"id"`
	Provider        string            `json:"provider"`
	DisplayName     string            `json:"display_name"`
	Email           string            `json:"email"`
	AuthMode        string            `json:"auth_mode"`
	Status          string            `json:"status"`
	Tier            string            `json:"tier"`
	Weight          int               `json:"weight"`
	Priority        int               `json:"priority"`
	SupportsModels  []string          `json:"supports_models"`
	Meta            map[string]string `json:"meta,omitempty"`
	LastRefreshedAt time.Time         `json:"last_refreshed_at"`
}

type ProviderAccess struct {
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
}

type ServicePackage struct {
	ID                         string           `json:"id"`
	Name                       string           `json:"name"`
	Tier                       string           `json:"tier"`
	Description                string           `json:"description"`
	ProviderAccess             []ProviderAccess `json:"provider_access"`
	AllowCrossProviderFallback bool             `json:"allow_cross_provider_fallback"`
	DefaultConcurrency         int              `json:"default_concurrency"`
}

type ModelTarget struct {
	Provider      string `json:"provider"`
	UpstreamModel string `json:"upstream_model"`
	Priority      int    `json:"priority"`
}

type ModelAliasPolicy struct {
	Alias   string        `json:"alias"`
	Targets []ModelTarget `json:"targets"`
}

type AccountPoolPolicy struct {
	ID         string   `json:"id"`
	Alias      string   `json:"alias"`
	Provider   string   `json:"provider"`
	AccountIDs []string `json:"account_ids"`
	Strategy   string   `json:"strategy"`
}

type Subscription struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	PackageID   string    `json:"package_id"`
	Status      string    `json:"status"`
	StartsAt    time.Time `json:"starts_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AssignedBy  string    `json:"assigned_by"`
	Description string    `json:"description"`
}

type APIKey struct {
	ID         string    `json:"id"`
	Key        string    `json:"key"`
	UserID     string    `json:"user_id"`
	PackageID  string    `json:"package_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type UsageLog struct {
	ID            string    `json:"id"`
	APIKeyID      string    `json:"api_key_id"`
	UserID        string    `json:"user_id"`
	ModelAlias    string    `json:"model_alias"`
	Provider      string    `json:"provider"`
	AccountID     string    `json:"account_id"`
	UpstreamModel string    `json:"upstream_model"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type Overview struct {
	Providers      map[string]int `json:"providers"`
	ActiveAccounts int            `json:"active_accounts"`
	ActiveKeys     int            `json:"active_keys"`
	ActiveSubs     int            `json:"active_subscriptions"`
	Packages       int            `json:"packages"`
	Users          int            `json:"users"`
}
