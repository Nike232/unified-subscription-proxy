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
	Products           []Product           `json:"products"`
	OAuthSessions      []OAuthSession      `json:"oauth_sessions,omitempty"`
	AuthSessions       []AuthSession       `json:"auth_sessions,omitempty"`
	Orders             []Order             `json:"orders,omitempty"`
	Payments           []Payment           `json:"payments,omitempty"`
	WebhookEvents      []WebhookEvent      `json:"webhook_events,omitempty"`
}

type Product struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`          // Enum: "top_up", "monthly_sub", "service"
	Price        float64 `json:"price"`
	AddBalance   float64 `json:"add_balance,omitempty"`
	AssignGroup  string  `json:"assign_group,omitempty"`
	DurationDays int     `json:"duration_days,omitempty"`
}

type User struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	Role         string  `json:"role"`
	PasswordHash string  `json:"password_hash,omitempty"`
	Balance      float64 `json:"balance"`
	Group        string  `json:"group"`
	TotalQuota   float64 `json:"total_quota"`
	RPM          int     `json:"rpm"`
	TPM          int     `json:"tpm"`
	Concurrency  int     `json:"concurrency"`
	Status       string  `json:"status"`
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
	DisplayName                string           `json:"display_name,omitempty"`
	Tier                       string           `json:"tier"`
	Description                string           `json:"description"`
	PriceCents                 int              `json:"price_cents,omitempty"`
	BillingCycle               string           `json:"billing_cycle,omitempty"`
	IsActive                   bool             `json:"is_active"`
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
	OrderID     string    `json:"order_id,omitempty"`
	Status      string    `json:"status"`
	StartsAt    time.Time `json:"starts_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AssignedBy  string    `json:"assigned_by"`
	Description string    `json:"description"`
	AutoRenew   bool      `json:"auto_renew,omitempty"`
	CancelledAt time.Time `json:"cancelled_at,omitempty"`
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
	ID                 string    `json:"id"`
	APIKeyID           string    `json:"api_key_id"`
	UserID             string    `json:"user_id"`
	ModelAlias         string    `json:"model_alias"`
	Provider           string    `json:"provider"`
	AccountID          string    `json:"account_id"`
	UpstreamModel      string    `json:"upstream_model"`
	Status             string    `json:"status"`
	RequestKind        string    `json:"request_kind,omitempty"`
	UpstreamStatusCode int       `json:"upstream_status_code,omitempty"`
	ErrorType          string    `json:"error_type,omitempty"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	PromptTokens       int       `json:"prompt_tokens,omitempty"`
	CompletionTokens   int       `json:"completion_tokens,omitempty"`
	TotalTokens        int       `json:"total_tokens,omitempty"`
	Cost               float64   `json:"cost,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type Overview struct {
	Providers      map[string]int `json:"providers"`
	ActiveAccounts int            `json:"active_accounts"`
	ActiveKeys     int            `json:"active_keys"`
	ActiveSubs     int            `json:"active_subscriptions"`
	Packages       int            `json:"packages"`
	Users          int            `json:"users"`
}

type OAuthSession struct {
	ID         string    `json:"id"`
	State      string    `json:"state"`
	AccountID  string    `json:"account_id"`
	Provider   string    `json:"provider"`
	CodeVerifier string  `json:"code_verifier,omitempty"`
	RedirectTo string    `json:"redirect_to,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type AuthSession struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Order struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	PackageID      string    `json:"package_id"`
	Status         string    `json:"status"`
	AmountCents    int       `json:"amount_cents"`
	Currency       string    `json:"currency"`
	BillingCycle   string    `json:"billing_cycle"`
	AutoRenew      bool      `json:"auto_renew,omitempty"`
	BindAPIKeyID   string    `json:"bind_api_key_id,omitempty"`
	CreateAPIKey   bool      `json:"create_api_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	PaymentID      string    `json:"payment_id,omitempty"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
}

type Payment struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	UserID        string    `json:"user_id"`
	Provider      string    `json:"provider"`
	Status        string    `json:"status"`
	AmountCents   int       `json:"amount_cents"`
	Currency      string    `json:"currency"`
	CheckoutURL   string    `json:"checkout_url,omitempty"`
	ProviderRef   string    `json:"provider_ref,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	WebhookStatus string    `json:"webhook_status,omitempty"`
}

type WebhookEvent struct {
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	EventType  string    `json:"event_type"`
	PaymentID  string    `json:"payment_id,omitempty"`
	Status     string    `json:"status,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
}
