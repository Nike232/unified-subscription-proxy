package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/store"
)

type Service struct {
	store store.Store
}

func New(st store.Store) *Service {
	return &Service{store: st}
}

func OverviewFromData(data domain.PlatformData) domain.Overview {
	out := domain.Overview{
		Providers: make(map[string]int),
		Packages:  len(data.ServicePackages),
		Users:     len(data.Users),
	}
	for _, acct := range data.UpstreamAccounts {
		out.Providers[acct.Provider]++
		if acct.Status == domain.AccountStatusActive {
			out.ActiveAccounts++
		}
	}
	for _, key := range data.APIKeys {
		if key.Status == "active" {
			out.ActiveKeys++
		}
	}
	for _, sub := range data.Subscriptions {
		if sub.Status == domain.SubscriptionStatusActive && sub.ExpiresAt.After(time.Now()) {
			out.ActiveSubs++
		}
	}
	return out
}

func (s *Service) Overview() (domain.Overview, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.Overview{}, err
	}
	return OverviewFromData(data), nil
}

func (s *Service) Data() (domain.PlatformData, error) {
	return s.store.Load()
}

func (s *Service) OAuthProviderSettings() (map[string]domain.OAuthProviderSetting, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if data.OAuthProviderSettings == nil {
		return map[string]domain.OAuthProviderSetting{}, nil
	}
	out := make(map[string]domain.OAuthProviderSetting, len(data.OAuthProviderSettings))
	for key, value := range data.OAuthProviderSettings {
		out[key] = value
	}
	return out, nil
}

func (s *Service) UpdateOAuthProviderSetting(provider string, patch map[string]any) (domain.OAuthProviderSetting, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return domain.OAuthProviderSetting{}, errors.New("provider is required")
	}
	var updated domain.OAuthProviderSetting
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		if data.OAuthProviderSettings == nil {
			data.OAuthProviderSettings = map[string]domain.OAuthProviderSetting{}
		}
		current := data.OAuthProviderSettings[provider]
		current.Provider = provider
		if v, ok := patch["client_id"].(string); ok {
			current.ClientID = strings.TrimSpace(v)
		}
		if v, ok := patch["client_secret"].(string); ok {
			current.ClientSecret = strings.TrimSpace(v)
		}
		if v, ok := patch["authorize_url"].(string); ok {
			current.AuthorizeURL = strings.TrimSpace(v)
		}
		if v, ok := patch["token_url"].(string); ok {
			current.TokenURL = strings.TrimSpace(v)
		}
		if v, ok := patch["redirect_url"].(string); ok {
			current.RedirectURL = strings.TrimSpace(v)
		}
		if v, ok := patch["prompt"].(string); ok {
			current.Prompt = strings.TrimSpace(v)
		}
		if v, ok := patch["access_type"].(string); ok {
			current.AccessType = strings.TrimSpace(v)
		}
		if v, ok := patch["use_pkce"].(bool); ok {
			current.UsePKCE = v
		}
		if v, ok := patch["include_granted_scopes"].(bool); ok {
			current.IncludeGrantedScopes = v
		}
		if v, ok := patch["scopes"].([]any); ok {
			current.Scopes = normalizeStringSlice(v)
		}
		if v, ok := patch["refresh_scopes"].([]any); ok {
			current.RefreshScopes = normalizeStringSlice(v)
		}
		if v, ok := patch["extra_authorize_params"].(map[string]any); ok {
			current.ExtraAuthorizeParams = map[string]string{}
			for key, raw := range v {
				if value, ok := raw.(string); ok {
					current.ExtraAuthorizeParams[strings.TrimSpace(key)] = strings.TrimSpace(value)
				}
			}
		}
		data.OAuthProviderSettings[provider] = current
		updated = current
		return nil
	})
	return updated, err
}

func (s *Service) ExplainDispatch(modelAlias, apiKey string) (DispatchTrace, error) {
	data, err := s.store.Load()
	if err != nil {
		return DispatchTrace{}, err
	}
	return ExplainDispatchInData(data, modelAlias, apiKey)
}

func (s *Service) AddUpstreamAccount(acct domain.UpstreamAccount) (domain.UpstreamAccount, error) {
	acct.Provider = strings.TrimSpace(strings.ToLower(acct.Provider))
	acct.DisplayName = strings.TrimSpace(acct.DisplayName)
	acct.Email = strings.TrimSpace(acct.Email)
	acct.Tier = strings.TrimSpace(acct.Tier)
	if strings.TrimSpace(acct.Provider) == "" {
		return domain.UpstreamAccount{}, errors.New("provider is required")
	}
	if strings.TrimSpace(acct.DisplayName) == "" {
		return domain.UpstreamAccount{}, errors.New("display_name is required")
	}
	if acct.Meta == nil {
		acct.Meta = map[string]string{}
	}
	acct = applyProviderDefaults(acct)
	if acct.ID == "" {
		acct.ID = "acct-" + randomID(4)
	}
	if acct.Status == "" {
		acct.Status = domain.AccountStatusActive
	}
	if acct.Weight == 0 {
		acct.Weight = 10
	}
	acct.LastRefreshedAt = time.Now().UTC()

	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		data.UpstreamAccounts = append(data.UpstreamAccounts, acct)
		return nil
	})
	return acct, err
}

func normalizeStringSlice(raw []any) []string {
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func applyProviderDefaults(acct domain.UpstreamAccount) domain.UpstreamAccount {
	acct.AuthMode = "oauth"
	acct.SupportsModels = providerDefaultModels(acct.Provider)
	if acct.Meta == nil {
		acct.Meta = map[string]string{}
	}
	if strings.TrimSpace(acct.Meta["base_url"]) == "" {
		if baseURL := providerDefaultBaseURL(acct.Provider); baseURL != "" {
			acct.Meta["base_url"] = baseURL
		}
	}
	return acct
}

func providerDefaultModels(provider string) []string {
	switch provider {
	case domain.ProviderOpenAI:
		return []string{"gpt-5", "gpt-4.1"}
	case domain.ProviderGemini:
		return []string{"gemini-2.5-pro", "gemini-2.5-flash"}
	case domain.ProviderClaude:
		return []string{"claude-sonnet-4.5", "claude-opus-4.1"}
	case domain.ProviderCodex:
		return []string{"gpt-5-codex", "gpt-5"}
	case domain.ProviderAntigravity:
		return []string{"hybrid-premium"}
	default:
		return nil
	}
}

func providerDefaultBaseURL(provider string) string {
	switch provider {
	case domain.ProviderOpenAI, domain.ProviderCodex:
		return "https://api.openai.com"
	case domain.ProviderGemini:
		return "https://generativelanguage.googleapis.com"
	case domain.ProviderClaude:
		return "https://api.anthropic.com"
	case domain.ProviderAntigravity:
		return "https://api.antigravity.example"
	default:
		return ""
	}
}

func (s *Service) UpdateUser(id string, patch map[string]any) (domain.User, error) {
	var updated domain.User
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.Users {
			if data.Users[i].ID != id {
				continue
			}
			if v, ok := patch["status"].(string); ok && strings.TrimSpace(v) != "" {
				data.Users[i].Status = v
			}
			if v, ok := patch["role"].(string); ok && strings.TrimSpace(v) != "" {
				data.Users[i].Role = v
			}
			if v, ok := patch["email"].(string); ok && strings.TrimSpace(v) != "" {
				data.Users[i].Email = strings.TrimSpace(v)
			}
			if v, ok := patch["name"].(string); ok && strings.TrimSpace(v) != "" {
				data.Users[i].Name = strings.TrimSpace(v)
			}
			if v, ok := patch["password"].(string); ok && strings.TrimSpace(v) != "" {
				data.Users[i].PasswordHash = v
			}
			if v, ok := patch["balance"].(float64); ok {
				data.Users[i].Balance = v
			}
			if v, ok := patch["concurrency"].(float64); ok {
				data.Users[i].Concurrency = int(v)
			}
			updated = data.Users[i]
			return nil
		}
		return errors.New("user not found")
	})
	return updated, err
}

func (s *Service) UpdateUpstreamAccount(id string, patch map[string]any) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != id {
				continue
			}
			if v, ok := patch["status"].(string); ok && strings.TrimSpace(v) != "" {
				data.UpstreamAccounts[i].Status = v
			}
			if v, ok := patch["tier"].(string); ok && strings.TrimSpace(v) != "" {
				data.UpstreamAccounts[i].Tier = v
			}
			if v, ok := patch["display_name"].(string); ok && strings.TrimSpace(v) != "" {
				data.UpstreamAccounts[i].DisplayName = v
			}
			if v, ok := patch["email"].(string); ok {
				data.UpstreamAccounts[i].Email = strings.TrimSpace(v)
			}
			if v, ok := patch["auth_mode"].(string); ok && strings.TrimSpace(v) != "" {
				data.UpstreamAccounts[i].AuthMode = "oauth"
			}
			if v, ok := patch["priority"].(float64); ok {
				data.UpstreamAccounts[i].Priority = int(v)
			}
			if v, ok := patch["weight"].(float64); ok && int(v) > 0 {
				data.UpstreamAccounts[i].Weight = int(v)
			}
			if v, ok := patch["meta"].(map[string]any); ok {
				data.UpstreamAccounts[i].Meta = map[string]string{}
				for key, raw := range v {
					if value, ok := raw.(string); ok {
						data.UpstreamAccounts[i].Meta[key] = value
					}
				}
			}
			data.UpstreamAccounts[i] = applyProviderDefaults(data.UpstreamAccounts[i])
			data.UpstreamAccounts[i].LastRefreshedAt = time.Now().UTC()
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) AddPackage(pkg domain.ServicePackage) (domain.ServicePackage, error) {
	if pkg.ID == "" {
		pkg.ID = "pkg-" + randomID(4)
	}
	if pkg.DefaultConcurrency <= 0 {
		pkg.DefaultConcurrency = 1
	}
	if strings.TrimSpace(pkg.DisplayName) == "" {
		pkg.DisplayName = pkg.Name
	}
	if strings.TrimSpace(pkg.BillingCycle) == "" {
		pkg.BillingCycle = "monthly"
	}
	if !pkg.IsActive {
		pkg.IsActive = true
	}
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		data.ServicePackages = append(data.ServicePackages, pkg)
		return nil
	})
	return pkg, err
}

func (s *Service) AssignSubscription(sub domain.Subscription) (domain.Subscription, error) {
	if sub.ID == "" {
		sub.ID = "sub-" + randomID(4)
	}
	if sub.Status == "" {
		sub.Status = domain.SubscriptionStatusActive
	}
	if sub.StartsAt.IsZero() {
		sub.StartsAt = time.Now().UTC()
	}
	if sub.ExpiresAt.IsZero() {
		sub.ExpiresAt = sub.StartsAt.Add(30 * 24 * time.Hour)
	}
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		data.Subscriptions = append(data.Subscriptions, sub)
		return nil
	})
	return sub, err
}

func (s *Service) CreateAPIKey(key domain.APIKey) (domain.APIKey, error) {
	if key.ID == "" {
		key.ID = "key-" + randomID(4)
	}
	if key.Key == "" {
		key.Key = "usp_" + randomID(12)
	}
	if key.Status == "" {
		key.Status = "active"
	}
	key.CreatedAt = time.Now().UTC()
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		data.APIKeys = append(data.APIKeys, key)
		return nil
	})
	return key, err
}

func (s *Service) ValidateAPIKey(rawKey string) (domain.APIKey, domain.ServicePackage, domain.User, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, err
	}
	return ValidateAPIKeyInData(data, rawKey)
}

func ValidateAPIKeyInData(data domain.PlatformData, rawKey string) (domain.APIKey, domain.ServicePackage, domain.User, error) {
	for _, key := range data.APIKeys {
		if key.Key != rawKey || key.Status != "active" {
			continue
		}

		var foundUser domain.User
		for _, u := range data.Users {
			if u.ID == key.UserID {
				foundUser = u
				if u.Status == "disabled" {
					return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, errors.New("user account disabled")
				}
				if u.Balance < 0 && u.Group != "gemini_vip" {
					return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, errors.New("insufficient balance")
				}
				if u.TotalQuota < 0 {
					return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, errors.New("insufficient quota")
				}
				break
			}
		}
		if foundUser.ID == "" {
			return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, errors.New("user not found")
		}

		if !hasActiveSubscription(data, key.UserID, key.PackageID) {
			return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, errors.New("subscription inactive or expired for api key")
		}
		for _, pkg := range data.ServicePackages {
			if pkg.ID == key.PackageID {
				return key, pkg, foundUser, nil
			}
		}
		return key, domain.ServicePackage{}, foundUser, errors.New("package not found for api key")
	}
	return domain.APIKey{}, domain.ServicePackage{}, domain.User{}, errors.New("invalid api key")
}

func (s *Service) ResolveDispatch(modelAlias, apiKey string) (DispatchResult, error) {
	data, err := s.store.Load()
	if err != nil {
		return DispatchResult{}, err
	}

	result, usageLog, err := ResolveDispatchInData(data, modelAlias, apiKey)
	if err != nil {
		return DispatchResult{}, err
	}

	_, _ = s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.APIKeys {
			if data.APIKeys[i].ID == result.APIKeyID {
				data.APIKeys[i].LastUsedAt = time.Now().UTC()
				break
			}
		}
		data.UsageLogs = append(data.UsageLogs, usageLog)
		return nil
	})

	return result, nil
}

func ResolveDispatchInData(data domain.PlatformData, modelAlias, apiKey string) (DispatchResult, domain.UsageLog, error) {
	trace, err := ExplainDispatchInData(data, modelAlias, apiKey)
	if err != nil {
		return DispatchResult{}, domain.UsageLog{}, err
	}
	selected := trace.Selected
	key := trace.APIKey
	result := DispatchResult{
		APIKeyID:       key.ID,
		PackageID:      key.PackageID,
		ModelAlias:     modelAlias,
		Provider:       selected.Provider,
		AccountID:      selected.AccountID,
		UpstreamModel:  selected.UpstreamModel,
		CandidateCount: len(trace.Candidates),
	}
	usageLog := domain.UsageLog{
		ID:            "log-" + randomID(6),
		APIKeyID:      key.ID,
		UserID:        key.UserID,
		ModelAlias:    modelAlias,
		Provider:      selected.Provider,
		AccountID:     selected.AccountID,
		UpstreamModel: selected.UpstreamModel,
		Status:        "dispatched",
		RequestKind:   "chat_completions",
		CreatedAt:     time.Now().UTC(),
	}
	return result, usageLog, nil
}

func ExplainDispatchInData(data domain.PlatformData, modelAlias, apiKey string) (DispatchTrace, error) {
	key, pkg, _, err := ValidateAPIKeyInData(data, apiKey)
	if err != nil {
		return DispatchTrace{}, err
	}

	allowedProviders := map[string]domain.ProviderAccess{}
	for _, access := range pkg.ProviderAccess {
		allowedProviders[access.Provider] = access
	}

	var policy *domain.ModelAliasPolicy
	for i := range data.ModelAliasPolicies {
		if data.ModelAliasPolicies[i].Alias == modelAlias {
			policy = &data.ModelAliasPolicies[i]
			break
		}
	}
	if policy == nil {
		return DispatchTrace{}, errors.New("model alias not found")
	}

	targets := append([]domain.ModelTarget(nil), policy.Targets...)
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].Priority > targets[j].Priority
	})

	var candidates []Candidate
	now := time.Now().UTC()
	for _, target := range targets {
		access, ok := allowedProviders[target.Provider]
		if !ok || !contains(access.Models, modelAlias) {
			continue
		}
		for _, acct := range data.UpstreamAccounts {
			if acct.Provider != target.Provider {
				continue
			}
			candidate := Candidate{
				AccountID:     acct.ID,
				Provider:      acct.Provider,
				UpstreamModel: target.UpstreamModel,
				Priority:      acct.Priority + target.Priority,
				Weight:        acct.Weight,
				AccountStatus: acct.Status,
			}
			if expiresAt, ok := AccountExpiresAt(acct); ok {
				candidate.ExpiresAt = expiresAt
			}
			if cooldownUntil, ok := AccountCooldownUntil(acct); ok {
				candidate.CooldownUntil = cooldownUntil
			}
			candidate.CanRefresh = AccountCanRefresh(acct)
			candidate.ConsecutiveFails = AccountConsecutiveFailures(acct)
			candidate.LastRefreshError = strings.TrimSpace(acct.Meta["last_refresh_error"])
			candidate.LastFailureReason = strings.TrimSpace(acct.Meta["last_failure_reason"])
			switch {
			case acct.Status == domain.AccountStatusDisabled:
				candidate.SkipReason = "disabled"
			case acct.Status == domain.AccountStatusInvalid:
				candidate.SkipReason = "invalid"
			case AccountInCooldown(acct, now):
				candidate.SkipReason = "cooldown"
			case AccountExpired(acct, now) && !candidate.CanRefresh:
				candidate.SkipReason = "expired"
			case !contains(acct.SupportsModels, target.UpstreamModel):
				candidate.SkipReason = "model_not_supported"
			}
			candidates = append(candidates, candidate)
		}
		if len(candidates) > 0 && !pkg.AllowCrossProviderFallback {
			break
		}
	}

	if len(candidates) == 0 {
		return DispatchTrace{}, errors.New("no available upstream account for alias under current package")
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			return candidates[i].Weight > candidates[j].Weight
		}
		return candidates[i].Priority > candidates[j].Priority
	})

	selected := Candidate{}
	foundSelected := false
	for _, candidate := range candidates {
		if candidate.SkipReason == "" {
			selected = candidate
			foundSelected = true
			break
		}
	}
	if !foundSelected {
		return DispatchTrace{}, errors.New("no available upstream account for alias under current package")
	}

	return DispatchTrace{
		ModelAlias: modelAlias,
		APIKey:     key,
		Package:    pkg,
		Candidates: candidates,
		Selected:   selected,
	}, nil
}

func (s *Service) AppendUsageLog(log domain.UsageLog) error {
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.APIKeys {
			if data.APIKeys[i].ID == log.APIKeyID {
				data.APIKeys[i].LastUsedAt = log.CreatedAt
				break
			}
		}

		var userGroup string
		var userIdx int = -1
		for i := range data.Users {
			if data.Users[i].ID == log.UserID {
				userIdx = i
				userGroup = data.Users[i].Group
				break
			}
		}

		// Subscription group rules
		if userGroup == "gemini_vip" && strings.HasPrefix(strings.ToLower(log.ModelAlias), "gemini") {
			log.Cost = 0 // Zero deduct for gemini monthly passes
		} else if log.Cost == 0 && log.TotalTokens > 0 {
			// Mock cost calculation: $0.002 per 1000 tokens
			log.Cost = float64(log.TotalTokens) * 0.000002
		}

		if userIdx >= 0 {
			if log.Cost > 0 {
				data.Users[userIdx].Balance -= log.Cost
			}
			if log.TotalTokens > 0 {
				// Deduct tokens from TotalQuota
				data.Users[userIdx].TotalQuota -= float64(log.TotalTokens)
			}
		}
		data.UsageLogs = append(data.UsageLogs, log)
		return nil
	})
	return err
}

func (s *Service) ListUsageLogs(provider, alias, status string) ([]domain.UsageLog, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]domain.UsageLog, 0)
	for i := len(data.UsageLogs) - 1; i >= 0; i-- {
		logEntry := data.UsageLogs[i]
		if provider != "" && logEntry.Provider != provider {
			continue
		}
		if alias != "" && logEntry.ModelAlias != alias {
			continue
		}
		if status != "" && logEntry.Status != status {
			continue
		}
		out = append(out, logEntry)
	}
	return out, nil
}

func (s *Service) ListUsageLogsFiltered(provider, alias, status, errorType, accountID string) ([]domain.UsageLog, error) {
	data, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]domain.UsageLog, 0)
	for i := len(data.UsageLogs) - 1; i >= 0; i-- {
		logEntry := data.UsageLogs[i]
		if provider != "" && logEntry.Provider != provider {
			continue
		}
		if alias != "" && logEntry.ModelAlias != alias {
			continue
		}
		if status != "" && logEntry.Status != status {
			continue
		}
		if errorType != "" && logEntry.ErrorType != errorType {
			continue
		}
		if accountID != "" && logEntry.AccountID != accountID {
			continue
		}
		out = append(out, logEntry)
	}
	return out, nil
}

type Candidate struct {
	AccountID         string    `json:"account_id"`
	Provider          string    `json:"provider"`
	UpstreamModel     string    `json:"upstream_model"`
	Priority          int       `json:"priority"`
	Weight            int       `json:"weight"`
	AccountStatus     string    `json:"account_status,omitempty"`
	ExpiresAt         time.Time `json:"expires_at,omitempty"`
	CooldownUntil     time.Time `json:"cooldown_until,omitempty"`
	CanRefresh        bool      `json:"can_refresh,omitempty"`
	ConsecutiveFails  int       `json:"consecutive_failures,omitempty"`
	LastRefreshError  string    `json:"last_refresh_error,omitempty"`
	LastFailureReason string    `json:"last_failure_reason,omitempty"`
	SkipReason        string    `json:"skip_reason,omitempty"`
}

type DispatchTrace struct {
	ModelAlias string                `json:"model_alias"`
	APIKey     domain.APIKey         `json:"api_key"`
	Package    domain.ServicePackage `json:"package"`
	Candidates []Candidate           `json:"candidates"`
	Selected   Candidate             `json:"selected"`
}

type DispatchResult struct {
	APIKeyID       string `json:"api_key_id"`
	PackageID      string `json:"package_id"`
	ModelAlias     string `json:"model_alias"`
	Provider       string `json:"provider"`
	AccountID      string `json:"account_id"`
	UpstreamModel  string `json:"upstream_model"`
	CandidateCount int    `json:"candidate_count"`
}

func randomID(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
