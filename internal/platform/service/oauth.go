package service

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type OAuthProviderConfig struct {
	Provider     string
	ClientID     string
	ClientSecret string
	AuthorizeURL string
	TokenURL     string
	RedirectURL  string
	Scopes       []string
}

type TokenPayload struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type AccountHealthPolicy struct {
	FailureThreshold int
	Cooldown         time.Duration
}

func (s *Service) CreateOAuthSession(accountID, redirectTo string) (domain.OAuthSession, string, error) {
	data, err := s.store.Load()
	if err != nil {
		return domain.OAuthSession{}, "", err
	}
	account, err := findAccountByID(data.UpstreamAccounts, accountID)
	if err != nil {
		return domain.OAuthSession{}, "", err
	}
	if !supportsOAuthProvider(account.Provider) {
		return domain.OAuthSession{}, "", fmt.Errorf("oauth not supported for provider %s", account.Provider)
	}
	session := domain.OAuthSession{
		ID:         "oauth-" + randomID(6),
		State:      randomID(16),
		AccountID:  account.ID,
		Provider:   account.Provider,
		RedirectTo: strings.TrimSpace(redirectTo),
		Status:     "pending",
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
	}
	_, err = s.store.Mutate(func(data *domain.PlatformData) error {
		data.OAuthSessions = append(data.OAuthSessions, session)
		return nil
	})
	return session, account.Provider, err
}

func BuildOAuthAuthorizeURL(cfg OAuthProviderConfig, state string) (string, error) {
	if strings.TrimSpace(cfg.AuthorizeURL) == "" || strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.RedirectURL) == "" {
		return "", errors.New("oauth provider authorize config incomplete")
	}
	u, err := url.Parse(cfg.AuthorizeURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("response_type", "code")
	query.Set("client_id", cfg.ClientID)
	query.Set("redirect_uri", cfg.RedirectURL)
	query.Set("state", state)
	if len(cfg.Scopes) > 0 {
		query.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (s *Service) CompleteOAuthSession(state string, token TokenPayload) (domain.UpstreamAccount, domain.OAuthSession, error) {
	var updated domain.UpstreamAccount
	var completed domain.OAuthSession
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		sessionIndex := -1
		for i := range data.OAuthSessions {
			if data.OAuthSessions[i].State == state {
				sessionIndex = i
				break
			}
		}
		if sessionIndex == -1 {
			return errors.New("oauth session not found")
		}
		session := data.OAuthSessions[sessionIndex]
		if session.ExpiresAt.Before(time.Now().UTC()) {
			data.OAuthSessions[sessionIndex].Status = "expired"
			return errors.New("oauth session expired")
		}
		accountIndex := -1
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID == session.AccountID {
				accountIndex = i
				break
			}
		}
		if accountIndex == -1 {
			return errors.New("account not found")
		}
		account := &data.UpstreamAccounts[accountIndex]
		ensureAccountMeta(account)
		account.Meta["access_token"] = token.AccessToken
		account.Meta["refresh_token"] = token.RefreshToken
		account.Meta["expires_at"] = token.ExpiresAt.UTC().Format(time.RFC3339)
		account.Meta["last_refresh_at"] = time.Now().UTC().Format(time.RFC3339)
		account.Meta["last_refresh_error"] = ""
		account.Meta["last_success_at"] = time.Now().UTC().Format(time.RFC3339)
		account.Meta["last_failure_reason"] = ""
		account.Meta["last_healthcheck_error"] = ""
		account.Meta["consecutive_failures"] = "0"
		delete(account.Meta, "cooldown_until")
		account.Status = domain.AccountStatusActive
		account.LastRefreshedAt = time.Now().UTC()

		data.OAuthSessions[sessionIndex].Status = "completed"
		updated = *account
		completed = data.OAuthSessions[sessionIndex]
		return nil
	})
	return updated, completed, err
}

func (s *Service) MarkOAuthSessionFailed(state, message string) error {
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.OAuthSessions {
			if data.OAuthSessions[i].State == state {
				data.OAuthSessions[i].Status = "failed"
				return nil
			}
		}
		return errors.New("oauth session not found")
	})
	return err
}

func (s *Service) RefreshAccountTokens(accountID string, token TokenPayload) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			data.UpstreamAccounts[i].Meta["access_token"] = token.AccessToken
			if token.RefreshToken != "" {
				data.UpstreamAccounts[i].Meta["refresh_token"] = token.RefreshToken
			}
			data.UpstreamAccounts[i].Meta["expires_at"] = token.ExpiresAt.UTC().Format(time.RFC3339)
			data.UpstreamAccounts[i].Meta["last_refresh_at"] = time.Now().UTC().Format(time.RFC3339)
			data.UpstreamAccounts[i].Meta["last_refresh_error"] = ""
			data.UpstreamAccounts[i].Meta["last_failure_reason"] = ""
			data.UpstreamAccounts[i].Meta["last_healthcheck_error"] = ""
			data.UpstreamAccounts[i].Meta["consecutive_failures"] = "0"
			delete(data.UpstreamAccounts[i].Meta, "cooldown_until")
			data.UpstreamAccounts[i].Status = domain.AccountStatusActive
			data.UpstreamAccounts[i].LastRefreshedAt = time.Now().UTC()
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) MarkAccountRefreshFailure(accountID, message string) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			data.UpstreamAccounts[i].Meta["last_refresh_error"] = message
			data.UpstreamAccounts[i].Meta["last_failure_at"] = time.Now().UTC().Format(time.RFC3339)
			data.UpstreamAccounts[i].Meta["last_failure_reason"] = message
			data.UpstreamAccounts[i].LastRefreshedAt = time.Now().UTC()
			data.UpstreamAccounts[i].Status = domain.AccountStatusInvalid
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) MarkAccountInvalid(accountID, message string) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			data.UpstreamAccounts[i].Meta["last_refresh_error"] = message
			data.UpstreamAccounts[i].Meta["last_failure_at"] = time.Now().UTC().Format(time.RFC3339)
			data.UpstreamAccounts[i].Meta["last_failure_reason"] = message
			data.UpstreamAccounts[i].LastRefreshedAt = time.Now().UTC()
			data.UpstreamAccounts[i].Status = domain.AccountStatusInvalid
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) RecordHealthCheck(accountID string, ok bool, message string) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			now := time.Now().UTC()
			data.UpstreamAccounts[i].Meta["last_healthcheck_at"] = now.Format(time.RFC3339)
			if ok {
				data.UpstreamAccounts[i].Meta["last_healthcheck_error"] = ""
				data.UpstreamAccounts[i].Meta["last_success_at"] = now.Format(time.RFC3339)
			} else {
				data.UpstreamAccounts[i].Meta["last_healthcheck_error"] = strings.TrimSpace(message)
				data.UpstreamAccounts[i].Meta["last_failure_at"] = now.Format(time.RFC3339)
				data.UpstreamAccounts[i].Meta["last_failure_reason"] = strings.TrimSpace(message)
			}
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) MarkAccountHealthy(accountID string) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			now := time.Now().UTC()
			data.UpstreamAccounts[i].Meta["last_success_at"] = now.Format(time.RFC3339)
			data.UpstreamAccounts[i].Meta["last_failure_reason"] = ""
			data.UpstreamAccounts[i].Meta["last_healthcheck_error"] = ""
			data.UpstreamAccounts[i].Meta["consecutive_failures"] = "0"
			delete(data.UpstreamAccounts[i].Meta, "cooldown_until")
			if data.UpstreamAccounts[i].Status != domain.AccountStatusDisabled {
				data.UpstreamAccounts[i].Status = domain.AccountStatusActive
			}
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) RestoreAccount(accountID string) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			data.UpstreamAccounts[i].Status = domain.AccountStatusActive
			data.UpstreamAccounts[i].Meta["last_refresh_error"] = ""
			data.UpstreamAccounts[i].Meta["last_failure_reason"] = ""
			data.UpstreamAccounts[i].Meta["last_healthcheck_error"] = ""
			data.UpstreamAccounts[i].Meta["consecutive_failures"] = "0"
			delete(data.UpstreamAccounts[i].Meta, "cooldown_until")
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) ClearAccountCooldown(accountID string) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			ensureAccountMeta(&data.UpstreamAccounts[i])
			delete(data.UpstreamAccounts[i].Meta, "cooldown_until")
			data.UpstreamAccounts[i].Meta["consecutive_failures"] = "0"
			updated = data.UpstreamAccounts[i]
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func (s *Service) CleanupOAuthSessions(now time.Time) (int, error) {
	removed := 0
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		kept := make([]domain.OAuthSession, 0, len(data.OAuthSessions))
		for _, session := range data.OAuthSessions {
			if session.ExpiresAt.Before(now) || session.Status == "completed" || session.Status == "failed" || session.Status == "expired" {
				removed++
				continue
			}
			kept = append(kept, session)
		}
		data.OAuthSessions = kept
		return nil
	})
	return removed, err
}

func (s *Service) RecordUsageOutcome(accountID, errorType, message string, policy AccountHealthPolicy) (domain.UpstreamAccount, error) {
	var updated domain.UpstreamAccount
	if policy.FailureThreshold <= 0 {
		policy.FailureThreshold = 3
	}
	_, err := s.store.Mutate(func(data *domain.PlatformData) error {
		for i := range data.UpstreamAccounts {
			if data.UpstreamAccounts[i].ID != accountID {
				continue
			}
			acct := &data.UpstreamAccounts[i]
			ensureAccountMeta(acct)
			now := time.Now().UTC()
			switch strings.TrimSpace(errorType) {
			case "":
				acct.Meta["last_success_at"] = now.Format(time.RFC3339)
				acct.Meta["last_failure_reason"] = ""
				acct.Meta["last_healthcheck_error"] = ""
				acct.Meta["consecutive_failures"] = "0"
				delete(acct.Meta, "cooldown_until")
				if acct.Status != domain.AccountStatusDisabled {
					acct.Status = domain.AccountStatusActive
				}
			case "auth_failed":
				acct.Meta["last_failure_at"] = now.Format(time.RFC3339)
				acct.Meta["last_failure_reason"] = strings.TrimSpace(message)
				acct.Meta["last_refresh_error"] = strings.TrimSpace(message)
				acct.Status = domain.AccountStatusInvalid
			case "quota_exceeded":
				acct.Meta["last_failure_at"] = now.Format(time.RFC3339)
				acct.Meta["last_failure_reason"] = strings.TrimSpace(message)
				acct.Meta["consecutive_failures"] = strconv.Itoa(AccountConsecutiveFailures(*acct) + 1)
			case "upstream_unavailable", "provider_error":
				failures := AccountConsecutiveFailures(*acct) + 1
				acct.Meta["consecutive_failures"] = strconv.Itoa(failures)
				acct.Meta["last_failure_at"] = now.Format(time.RFC3339)
				acct.Meta["last_failure_reason"] = strings.TrimSpace(message)
				if failures >= policy.FailureThreshold && policy.Cooldown > 0 {
					acct.Meta["cooldown_until"] = now.Add(policy.Cooldown).Format(time.RFC3339)
				}
			default:
				acct.Meta["last_failure_at"] = now.Format(time.RFC3339)
				acct.Meta["last_failure_reason"] = strings.TrimSpace(message)
			}
			updated = *acct
			return nil
		}
		return errors.New("account not found")
	})
	return updated, err
}

func RefreshableProvider(provider string) bool {
	return supportsOAuthProvider(provider)
}

func supportsOAuthProvider(provider string) bool {
	switch provider {
	case domain.ProviderClaude, domain.ProviderCodex, domain.ProviderAntigravity:
		return true
	default:
		return false
	}
}

func findAccountByID(accounts []domain.UpstreamAccount, accountID string) (domain.UpstreamAccount, error) {
	for _, account := range accounts {
		if account.ID == accountID {
			return account, nil
		}
	}
	return domain.UpstreamAccount{}, errors.New("account not found")
}

func AccountExpiresAt(account domain.UpstreamAccount) (time.Time, bool) {
	raw := strings.TrimSpace(account.Meta["expires_at"])
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func AccountCanRefresh(account domain.UpstreamAccount) bool {
	if !supportsOAuthProvider(account.Provider) {
		return false
	}
	return strings.TrimSpace(account.Meta["refresh_token"]) != ""
}

func AccountConsecutiveFailures(account domain.UpstreamAccount) int {
	raw := strings.TrimSpace(account.Meta["consecutive_failures"])
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func AccountCooldownUntil(account domain.UpstreamAccount) (time.Time, bool) {
	raw := strings.TrimSpace(account.Meta["cooldown_until"])
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func AccountInCooldown(account domain.UpstreamAccount, now time.Time) bool {
	until, ok := AccountCooldownUntil(account)
	if !ok {
		return false
	}
	return until.After(now)
}

func AccountExpired(account domain.UpstreamAccount, now time.Time) bool {
	expiresAt, ok := AccountExpiresAt(account)
	if !ok {
		return false
	}
	return !expiresAt.After(now)
}

func AccountExpiringSoon(account domain.UpstreamAccount, now time.Time, window time.Duration) bool {
	expiresAt, ok := AccountExpiresAt(account)
	if !ok {
		return false
	}
	return expiresAt.Before(now.Add(window))
}

func ensureAccountMeta(account *domain.UpstreamAccount) {
	if account.Meta == nil {
		account.Meta = map[string]string{}
	}
}
