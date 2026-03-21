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

func (s *Service) Overview() (domain.Overview, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.Overview{}, err
	}
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
	return out, nil
}

func (s *Service) Data() (domain.PlatformData, error) {
	return s.store.Load()
}

func (s *Service) AddUpstreamAccount(acct domain.UpstreamAccount) (domain.UpstreamAccount, error) {
	if strings.TrimSpace(acct.Provider) == "" {
		return domain.UpstreamAccount{}, errors.New("provider is required")
	}
	if strings.TrimSpace(acct.DisplayName) == "" {
		return domain.UpstreamAccount{}, errors.New("display_name is required")
	}
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

func (s *Service) ValidateAPIKey(rawKey string) (domain.APIKey, domain.ServicePackage, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.APIKey{}, domain.ServicePackage{}, err
	}
	for _, key := range data.APIKeys {
		if key.Key != rawKey || key.Status != "active" {
			continue
		}
		for _, pkg := range data.ServicePackages {
			if pkg.ID == key.PackageID {
				return key, pkg, nil
			}
		}
		return key, domain.ServicePackage{}, errors.New("package not found for api key")
	}
	return domain.APIKey{}, domain.ServicePackage{}, errors.New("invalid api key")
}

func (s *Service) ResolveDispatch(modelAlias, apiKey string) (DispatchResult, error) {
	data, err := s.store.Load()
	if err != nil {
		return DispatchResult{}, err
	}

	key, pkg, err := s.ValidateAPIKey(apiKey)
	if err != nil {
		return DispatchResult{}, err
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
		return DispatchResult{}, errors.New("model alias not found")
	}

	targets := append([]domain.ModelTarget(nil), policy.Targets...)
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].Priority > targets[j].Priority
	})

	var candidates []Candidate
	for _, target := range targets {
		access, ok := allowedProviders[target.Provider]
		if !ok || !contains(access.Models, modelAlias) {
			continue
		}
		for _, acct := range data.UpstreamAccounts {
			if acct.Provider != target.Provider || acct.Status != domain.AccountStatusActive {
				continue
			}
			if !contains(acct.SupportsModels, target.UpstreamModel) {
				continue
			}
			candidates = append(candidates, Candidate{
				AccountID:     acct.ID,
				Provider:      acct.Provider,
				UpstreamModel: target.UpstreamModel,
				Priority:      acct.Priority + target.Priority,
				Weight:        acct.Weight,
			})
		}
		if len(candidates) > 0 && !pkg.AllowCrossProviderFallback {
			break
		}
	}

	if len(candidates) == 0 {
		return DispatchResult{}, errors.New("no available upstream account for alias under current package")
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			return candidates[i].Weight > candidates[j].Weight
		}
		return candidates[i].Priority > candidates[j].Priority
	})

	selected := candidates[0]
	result := DispatchResult{
		APIKeyID:       key.ID,
		PackageID:      pkg.ID,
		ModelAlias:     modelAlias,
		Provider:       selected.Provider,
		AccountID:      selected.AccountID,
		UpstreamModel:  selected.UpstreamModel,
		CandidateCount: len(candidates),
	}

	_, _ = s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.APIKeys {
			if data.APIKeys[i].ID == key.ID {
				data.APIKeys[i].LastUsedAt = time.Now().UTC()
				break
			}
		}
		data.UsageLogs = append(data.UsageLogs, domain.UsageLog{
			ID:            "log-" + randomID(6),
			APIKeyID:      key.ID,
			UserID:        key.UserID,
			ModelAlias:    modelAlias,
			Provider:      selected.Provider,
			AccountID:     selected.AccountID,
			UpstreamModel: selected.UpstreamModel,
			Status:        "dispatched",
			CreatedAt:     time.Now().UTC(),
		})
		return nil
	})

	return result, nil
}

type Candidate struct {
	AccountID     string `json:"account_id"`
	Provider      string `json:"provider"`
	UpstreamModel string `json:"upstream_model"`
	Priority      int    `json:"priority"`
	Weight        int    `json:"weight"`
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
