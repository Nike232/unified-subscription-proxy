package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/service"
	proxyproviders "unifiedsubscriptionproxy/internal/proxy/providers"
)

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    string `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

type automationConfig struct {
	Enabled                bool
	RefreshInterval        time.Duration
	HealthcheckInterval    time.Duration
	SessionCleanupInterval time.Duration
	FailureThreshold       int
	Cooldown               time.Duration
}

const geminiCLIBuiltinClientID = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
const geminiCLIBuiltinRedirectURL = "https://codeassist.google.com/authcode"
const openAIBuiltinRedirectURL = "http://localhost:1455/auth/callback"

func defaultOAuthCallbackURL(provider string) string {
	base := strings.TrimSpace(os.Getenv("CONTROL_PLANE_PUBLIC_ORIGIN"))
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return strings.TrimRight(base, "/") + "/api/admin/oauth/callback/" + provider
}

func oauthBaseConfigs() map[string]service.OAuthProviderConfig {
	geminiClientID := strings.TrimSpace(os.Getenv("GEMINI_OAUTH_CLIENT_ID"))
	geminiClientSecret := strings.TrimSpace(os.Getenv("GEMINI_OAUTH_CLIENT_SECRET"))
	geminiRedirectURL := strings.TrimSpace(os.Getenv("GEMINI_OAUTH_REDIRECT_URL"))
	if geminiClientID == "" {
		geminiClientID = geminiCLIBuiltinClientID
		if geminiClientSecret == "" {
			geminiClientSecret = strings.TrimSpace(os.Getenv("GEMINI_CLI_OAUTH_CLIENT_SECRET"))
		}
		if geminiRedirectURL == "" {
			geminiRedirectURL = geminiCLIBuiltinRedirectURL
		}
	}
	if geminiRedirectURL == "" {
		geminiRedirectURL = defaultOAuthCallbackURL(domain.ProviderGemini)
	}
	return map[string]service.OAuthProviderConfig{
		domain.ProviderOpenAI: {
			Provider:             domain.ProviderOpenAI,
			ClientID:             getenv("OPENAI_OAUTH_CLIENT_ID", "app_EMoamEEZ73f0CkXaXp7hrann"),
			ClientSecret:         os.Getenv("OPENAI_OAUTH_CLIENT_SECRET"),
			AuthorizeURL:         getenv("OPENAI_OAUTH_AUTHORIZE_URL", "https://auth.openai.com/oauth/authorize"),
			TokenURL:             getenv("OPENAI_OAUTH_TOKEN_URL", "https://auth.openai.com/oauth/token"),
			RedirectURL:          getenv("OPENAI_OAUTH_REDIRECT_URL", openAIBuiltinRedirectURL),
			Scopes:               defaultScopes(splitCSV(os.Getenv("OPENAI_OAUTH_SCOPES")), []string{"openid", "profile", "email", "offline_access"}),
			RefreshScopes:        defaultScopes(splitCSV(os.Getenv("OPENAI_OAUTH_REFRESH_SCOPES")), []string{"openid", "profile", "email"}),
			UsePKCE:              true,
			ExtraAuthorizeParams: map[string]string{"id_token_add_organizations": "true", "codex_cli_simplified_flow": "true"},
		},
		domain.ProviderGemini: {
			Provider:              domain.ProviderGemini,
			ClientID:              geminiClientID,
			ClientSecret:          geminiClientSecret,
			AuthorizeURL:          getenv("GEMINI_OAUTH_AUTHORIZE_URL", "https://accounts.google.com/o/oauth2/v2/auth"),
			TokenURL:              getenv("GEMINI_OAUTH_TOKEN_URL", "https://oauth2.googleapis.com/token"),
			RedirectURL:           geminiRedirectURL,
			Scopes:                defaultScopes(splitCSV(os.Getenv("GEMINI_OAUTH_SCOPES")), []string{"https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}),
			UsePKCE:               true,
			AccessType:            "offline",
			Prompt:                getenv("GEMINI_OAUTH_PROMPT", "consent"),
			IncludeGrantedScopes:  true,
		},
		domain.ProviderClaude: {
			Provider:     domain.ProviderClaude,
			ClientID:     os.Getenv("CLAUDE_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("CLAUDE_OAUTH_CLIENT_SECRET"),
			AuthorizeURL: os.Getenv("CLAUDE_OAUTH_AUTHORIZE_URL"),
			TokenURL:     os.Getenv("CLAUDE_OAUTH_TOKEN_URL"),
			RedirectURL:  os.Getenv("CLAUDE_OAUTH_REDIRECT_URL"),
			Scopes:       splitCSV(os.Getenv("CLAUDE_OAUTH_SCOPES")),
		},
		domain.ProviderCodex: {
			Provider:     domain.ProviderCodex,
			ClientID:     os.Getenv("CODEX_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("CODEX_OAUTH_CLIENT_SECRET"),
			AuthorizeURL: os.Getenv("CODEX_OAUTH_AUTHORIZE_URL"),
			TokenURL:     os.Getenv("CODEX_OAUTH_TOKEN_URL"),
			RedirectURL:  os.Getenv("CODEX_OAUTH_REDIRECT_URL"),
			Scopes:       splitCSV(os.Getenv("CODEX_OAUTH_SCOPES")),
		},
		domain.ProviderAntigravity: {
			Provider:     domain.ProviderAntigravity,
			ClientID:     os.Getenv("ANTIGRAVITY_OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET"),
			AuthorizeURL: os.Getenv("ANTIGRAVITY_OAUTH_AUTHORIZE_URL"),
			TokenURL:     os.Getenv("ANTIGRAVITY_OAUTH_TOKEN_URL"),
			RedirectURL:  os.Getenv("ANTIGRAVITY_OAUTH_REDIRECT_URL"),
			Scopes:       splitCSV(os.Getenv("ANTIGRAVITY_OAUTH_SCOPES")),
		},
	}
}

func oauthConfigs(svc *service.Service) map[string]service.OAuthProviderConfig {
	configs := oauthBaseConfigs()
	if svc == nil {
		return configs
	}
	settings, err := svc.OAuthProviderSettings()
	if err != nil {
		return configs
	}
	for provider, setting := range settings {
		cfg, ok := configs[provider]
		if !ok {
			cfg = service.OAuthProviderConfig{Provider: provider}
		}
		if strings.TrimSpace(setting.ClientID) != "" {
			cfg.ClientID = strings.TrimSpace(setting.ClientID)
		}
		if strings.TrimSpace(setting.ClientSecret) != "" {
			cfg.ClientSecret = strings.TrimSpace(setting.ClientSecret)
		}
		if strings.TrimSpace(setting.AuthorizeURL) != "" {
			cfg.AuthorizeURL = strings.TrimSpace(setting.AuthorizeURL)
		}
		if strings.TrimSpace(setting.TokenURL) != "" {
			cfg.TokenURL = strings.TrimSpace(setting.TokenURL)
		}
		if strings.TrimSpace(setting.RedirectURL) != "" {
			cfg.RedirectURL = strings.TrimSpace(setting.RedirectURL)
		}
		if len(setting.Scopes) > 0 {
			cfg.Scopes = append([]string(nil), setting.Scopes...)
		}
		if len(setting.RefreshScopes) > 0 {
			cfg.RefreshScopes = append([]string(nil), setting.RefreshScopes...)
		}
		if strings.TrimSpace(setting.Prompt) != "" {
			cfg.Prompt = strings.TrimSpace(setting.Prompt)
		}
		if strings.TrimSpace(setting.AccessType) != "" {
			cfg.AccessType = strings.TrimSpace(setting.AccessType)
		}
		if setting.UsePKCE {
			cfg.UsePKCE = true
		}
		if setting.IncludeGrantedScopes {
			cfg.IncludeGrantedScopes = true
		}
		if len(setting.ExtraAuthorizeParams) > 0 {
			cfg.ExtraAuthorizeParams = map[string]string{}
			for key, value := range setting.ExtraAuthorizeParams {
				cfg.ExtraAuthorizeParams[key] = value
			}
		}
		configs[provider] = cfg
	}
	return configs
}

func loadAutomationConfig() automationConfig {
	return automationConfig{
		Enabled:                strings.ToLower(getenv("CONTROL_PLANE_AUTOMATION_ENABLED", "true")) != "false",
		RefreshInterval:        parseDurationEnv("CONTROL_PLANE_REFRESH_INTERVAL", 2*time.Minute),
		HealthcheckInterval:    parseDurationEnv("CONTROL_PLANE_HEALTHCHECK_INTERVAL", 5*time.Minute),
		SessionCleanupInterval: parseDurationEnv("CONTROL_PLANE_SESSION_CLEANUP_INTERVAL", 10*time.Minute),
		FailureThreshold:       parseIntEnv("CONTROL_PLANE_FAILURE_THRESHOLD", 3),
		Cooldown:               parseDurationEnv("CONTROL_PLANE_ACCOUNT_COOLDOWN", 10*time.Minute),
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func defaultScopes(current []string, fallback []string) []string {
	if len(current) > 0 {
		return current
	}
	return append([]string(nil), fallback...)
}

func refreshExpiringAccounts(ctx context.Context, svc *service.Service, client *http.Client, configs map[string]service.OAuthProviderConfig, automation automationConfig) error {
	data, err := svc.Data()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, account := range data.UpstreamAccounts {
		if !service.RefreshableProvider(account.Provider) {
			continue
		}
		if !service.AccountExpiringSoon(account, now, 5*time.Minute) {
			continue
		}
		if !service.AccountCanRefresh(account) {
			continue
		}
		if _, err := refreshAccount(ctx, svc, client, account, configs, automation); err != nil {
			return nil
		}
	}
	return nil
}

func runAutomationWorkers(ctx context.Context, svc *service.Service, client *http.Client, registry *proxyproviders.Registry, loadConfigs func() map[string]service.OAuthProviderConfig, automation automationConfig) {
	if !automation.Enabled {
		return
	}

	startTickerWorker(ctx, automation.RefreshInterval, func() {
		_ = refreshExpiringAccounts(context.Background(), svc, client, loadConfigs(), automation)
	})
	startTickerWorker(ctx, automation.HealthcheckInterval, func() {
		_ = runHealthChecks(context.Background(), svc, registry)
	})
	startTickerWorker(ctx, automation.SessionCleanupInterval, func() {
		_, _ = svc.CleanupOAuthSessions(time.Now().UTC())
		_, _ = svc.CleanupAuthSessions(time.Now().UTC())
	})
}

func startTickerWorker(ctx context.Context, interval time.Duration, fn func()) {
	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		fn()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fn()
			}
		}
	}()
}

func runHealthChecks(ctx context.Context, svc *service.Service, registry *proxyproviders.Registry) error {
	data, err := svc.Data()
	if err != nil {
		return err
	}
	for _, account := range data.UpstreamAccounts {
		provider, err := registry.Provider(account.Provider)
		if err != nil {
			continue
		}
		result, err := provider.HealthCheck(ctx, account)
		if err != nil {
			_, _ = svc.RecordHealthCheck(account.ID, false, err.Error())
			continue
		}
		_, _ = svc.RecordHealthCheck(account.ID, result.OK, result.Message)
	}
	return nil
}

func refreshAccount(ctx context.Context, svc *service.Service, client *http.Client, account domain.UpstreamAccount, configs map[string]service.OAuthProviderConfig, automation automationConfig) (domain.UpstreamAccount, error) {
	cfg, ok := configs[account.Provider]
	if !ok {
		return domain.UpstreamAccount{}, errors.New("oauth config missing for provider")
	}
	refreshToken := strings.TrimSpace(account.Meta["refresh_token"])
	if refreshToken == "" {
		return domain.UpstreamAccount{}, errors.New("refresh token missing")
	}
	payload, err := exchangeOAuthToken(ctx, client, cfg, url.Values{
		"grant_type":    []string{"refresh_token"},
		"refresh_token": []string{refreshToken},
	})
	if err != nil {
		updated, _ := svc.MarkAccountRefreshFailure(account.ID, err.Error())
		_, _ = svc.RecordUsageOutcome(account.ID, "auth_failed", err.Error(), service.AccountHealthPolicy{
			FailureThreshold: automation.FailureThreshold,
			Cooldown:         automation.Cooldown,
		})
		return updated, err
	}
	return svc.RefreshAccountTokens(account.ID, payload)
}

func exchangeAuthorizationCode(ctx context.Context, client *http.Client, cfg service.OAuthProviderConfig, session domain.OAuthSession, code string) (service.TokenPayload, error) {
	values := url.Values{
		"grant_type":   []string{"authorization_code"},
		"code":         []string{code},
		"redirect_uri": []string{cfg.RedirectURL},
	}
	if cfg.UsePKCE && strings.TrimSpace(session.CodeVerifier) != "" {
		values.Set("code_verifier", session.CodeVerifier)
	}
	return exchangeOAuthToken(ctx, client, cfg, values)
}

func exchangeOAuthToken(ctx context.Context, client *http.Client, cfg service.OAuthProviderConfig, values url.Values) (service.TokenPayload, error) {
	if strings.TrimSpace(cfg.TokenURL) == "" {
		return service.TokenPayload{}, errors.New("oauth token url not configured")
	}
	if strings.TrimSpace(cfg.ClientID) != "" {
		values.Set("client_id", cfg.ClientID)
	}
	if strings.TrimSpace(cfg.ClientSecret) != "" {
		values.Set("client_secret", cfg.ClientSecret)
	}
	if values.Get("grant_type") == "refresh_token" && len(cfg.RefreshScopes) > 0 {
		values.Set("scope", strings.Join(cfg.RefreshScopes, " "))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return service.TokenPayload{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return service.TokenPayload{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return service.TokenPayload{}, err
	}
	if resp.StatusCode >= 400 {
		return service.TokenPayload{}, fmt.Errorf("token exchange failed: %s", strings.TrimSpace(string(body)))
	}
	var parsed oauthTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return service.TokenPayload{}, err
	}
	expiresAt := time.Now().UTC().Add(time.Hour)
	switch {
	case strings.TrimSpace(parsed.ExpiresAt) != "":
		if ts, err := time.Parse(time.RFC3339, parsed.ExpiresAt); err == nil {
			expiresAt = ts.UTC()
		}
	case parsed.ExpiresIn > 0:
		expiresAt = time.Now().UTC().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	}
	return service.TokenPayload{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func renderOAuthCallbackPage(account domain.UpstreamAccount, redirectTo string) []byte {
	message := fmt.Sprintf("OAuth login completed for %s (%s).", account.DisplayName, account.Provider)
	if strings.TrimSpace(redirectTo) != "" {
		return []byte(fmt.Sprintf(`<html><body><script>window.location.href=%q;</script><p>%s</p></body></html>`, redirectTo, message))
	}
	return []byte(fmt.Sprintf(`<html><body><p>%s</p></body></html>`, message))
}

func writeHTML(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.Copy(w, bytes.NewReader(body))
}

func mapsKeys(values map[string]service.OAuthProviderConfig) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
