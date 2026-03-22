package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/service"
	"unifiedsubscriptionproxy/internal/platform/store"
	proxyproviders "unifiedsubscriptionproxy/internal/proxy/providers"
)

func main() {
	addr := getenv("CONTROL_PLANE_ADDR", ":8080")
	publicOrigin := controlPlanePublicOrigin(addr)
	dataPath := getenv("PLATFORM_DATA_FILE", "./data/platform.json")
	storeBackend := getenv("CONTROL_PLANE_STORE_BACKEND", "file")
	databaseURL := getenv("DATABASE_URL", "")
	runtimeConfig := loadProxyRuntimeConfig()

	st, err := openConfiguredStoreWithRetry(context.Background(), storeBackend, dataPath, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	svc := service.New(st)
	providerRegistry := proxyproviders.NewRegistry(http.DefaultClient)
	loadOAuthProviderConfigs := func() map[string]service.OAuthProviderConfig {
		return oauthConfigs(svc)
	}
	oauthHTTPClient := http.DefaultClient
	automation := loadAutomationConfig()
	healthPolicy := service.AccountHealthPolicy{
		FailureThreshold: automation.FailureThreshold,
		Cooldown:         automation.Cooldown,
	}
	mux := http.NewServeMux()
	runAutomationWorkers(context.Background(), svc, oauthHTTPClient, providerRegistry, loadOAuthProviderConfigs, automation)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"service":       "control-plane",
			"store_backend": storeBackend,
			"proxy_runtime": runtimeConfig,
		})
	})

	mux.HandleFunc("/api/auth/login", loginHandler(svc))
	mux.HandleFunc("/api/auth/logout", logoutHandler(svc))
	mux.HandleFunc("/api/auth/me", meHandler(svc))

	mux.HandleFunc("/api/admin/overview", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		overview, err := svc.Overview()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, overview)
	}))

	mux.HandleFunc("/api/admin/kernel-status", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, probeKernelStatus(r.Context(), http.DefaultClient, runtimeConfig))
	}))

	mux.HandleFunc("/api/admin/oauth-configs", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		settings, err := svc.OAuthProviderSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		configs := loadOAuthProviderConfigs()
		writeJSON(w, http.StatusOK, map[string]any{"settings": settings, "effective": configs})
	}))

	mux.HandleFunc("/api/admin/oauth-configs/", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		provider := strings.TrimPrefix(r.URL.Path, "/api/admin/oauth-configs/")
		if provider == "" {
			writeError(w, http.StatusBadRequest, errString("missing provider"))
			return
		}
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		setting, err := svc.UpdateOAuthProviderSetting(provider, patch)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, setting)
	}))

	mux.HandleFunc("/api/admin/data", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = refreshExpiringAccounts(r.Context(), svc, oauthHTTPClient, loadOAuthProviderConfigs(), automation)
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		safeData := data
		for i := range safeData.Users {
			safeData.Users[i] = safeUser(safeData.Users[i])
		}
		writeJSON(w, http.StatusOK, safeData)
	}))

	mux.HandleFunc("/api/admin/users", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		safeUsers := make([]domain.User, len(data.Users))
		for i, u := range data.Users {
			safeUsers[i] = safeUser(u)
			safeUsers[i].Balance = u.Balance
			safeUsers[i].Concurrency = u.Concurrency
			safeUsers[i].Status = u.Status
		}
		writeJSON(w, http.StatusOK, safeUsers)
	}))

	mux.HandleFunc("/api/admin/users/", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errString("missing user id"))
			return
		}
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		out, err := svc.UpdateUser(id, patch)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, safeUser(out))
	}))

	mux.HandleFunc("/api/admin/upstream-accounts", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		switch r.Method {
		case http.MethodGet:
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, data.UpstreamAccounts)
		case http.MethodPost:
			var account domain.UpstreamAccount
			if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			out, err := svc.AddUpstreamAccount(account)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, out)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/api/admin/upstream-accounts/", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method == http.MethodOptions {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/admin/upstream-accounts/")
		if strings.HasSuffix(path, "/test") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimSuffix(path, "/test")
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			account, err := findAccount(data, id)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			provider, err := providerRegistry.Provider(account.Provider)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			result, err := provider.HealthCheck(r.Context(), account)
			if err != nil {
				_, _ = svc.RecordHealthCheck(account.ID, false, err.Error())
				writeError(w, http.StatusBadRequest, err)
				return
			}
			updatedAccount, _ := svc.RecordHealthCheck(account.ID, result.OK, result.Message)
			if updatedAccount.ID != "" {
				account = updatedAccount
			}
			writeJSON(w, http.StatusOK, map[string]any{"account": account, "health": result})
			return
		}
		if strings.HasSuffix(path, "/oauth/start") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimSuffix(path, "/oauth/start")
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			account, err := findAccount(data, id)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			cfg, ok := loadOAuthProviderConfigs()[account.Provider]
			if !ok {
				writeError(w, http.StatusBadRequest, errString("oauth provider config missing"))
				return
			}
			var req struct {
				RedirectTo string `json:"redirect_to"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			session, providerName, err := svc.CreateOAuthSession(id, req.RedirectTo)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			authURL, err := service.BuildOAuthAuthorizeURL(cfg, session)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"session":       session,
				"provider":      providerName,
				"authorize_url": authURL,
			})
			return
		}
		if strings.HasSuffix(path, "/oauth/complete") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimSuffix(path, "/oauth/complete")
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			account, err := findAccount(data, id)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			cfg, ok := loadOAuthProviderConfigs()[account.Provider]
			if !ok {
				writeError(w, http.StatusBadRequest, errString("oauth provider config missing"))
				return
			}
			var req struct {
				State       string `json:"state"`
				Code        string `json:"code"`
				CallbackURL string `json:"callback_url"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, errString("invalid oauth completion payload"))
				return
			}
			state := strings.TrimSpace(req.State)
			code := strings.TrimSpace(req.Code)
			if callbackURL := strings.TrimSpace(req.CallbackURL); callbackURL != "" {
				if parsed, err := url.Parse(callbackURL); err == nil {
					if state == "" {
						state = strings.TrimSpace(parsed.Query().Get("state"))
					}
					if code == "" {
						code = strings.TrimSpace(parsed.Query().Get("code"))
					}
				}
			}
			if state == "" || code == "" {
				writeError(w, http.StatusBadRequest, errString("missing oauth callback code or state"))
				return
			}
			session, err := svc.OAuthSessionByState(state)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if session.AccountID != account.ID {
				writeError(w, http.StatusBadRequest, errString("oauth session account mismatch"))
				return
			}
			tokenPayload, err := exchangeAuthorizationCode(r.Context(), oauthHTTPClient, cfg, session, code)
			if err != nil {
				_ = svc.MarkOAuthSessionFailed(state, err.Error())
				writeError(w, http.StatusBadRequest, err)
				return
			}
			updatedAccount, _, err := svc.CompleteOAuthSession(state, tokenPayload)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, updatedAccount)
			return
		}
		if strings.HasSuffix(path, "/refresh") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimSuffix(path, "/refresh")
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			account, err := findAccount(data, id)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			updated, err := refreshAccount(r.Context(), svc, oauthHTTPClient, account, loadOAuthProviderConfigs(), automation)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, updated)
			return
		}
		if strings.HasSuffix(path, "/recover") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimSuffix(path, "/recover")
			updated, err := svc.RestoreAccount(id)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, updated)
			return
		}
		if strings.HasSuffix(path, "/clear-cooldown") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			id := strings.TrimSuffix(path, "/clear-cooldown")
			updated, err := svc.ClearAccountCooldown(id)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, updated)
			return
		}
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		id := path
		if id == "" {
			writeError(w, http.StatusBadRequest, errString("missing account id"))
			return
		}
		var patch map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		out, err := svc.UpdateUpstreamAccount(id, patch)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}))

	mux.HandleFunc("/api/admin/packages", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		switch r.Method {
		case http.MethodGet:
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, data.ServicePackages)
		case http.MethodPost:
			var pkg domain.ServicePackage
			if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			out, err := svc.AddPackage(pkg)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, out)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/api/admin/subscriptions", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		switch r.Method {
		case http.MethodGet:
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, data.Subscriptions)
		case http.MethodPost:
			var sub domain.Subscription
			if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			out, err := svc.AssignSubscription(sub)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, out)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/api/admin/api-keys", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		switch r.Method {
		case http.MethodGet:
			data, err := svc.Data()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, data.APIKeys)
		case http.MethodPost:
			var key domain.APIKey
			if err := json.NewDecoder(r.Body).Decode(&key); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			out, err := svc.CreateAPIKey(key)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, out)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/api/admin/orders", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		orders, err := svc.ListOrdersFiltered(
			strings.TrimSpace(r.URL.Query().Get("user_id")),
			strings.TrimSpace(r.URL.Query().Get("status")),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, orders)
	}))

	mux.HandleFunc("/api/admin/orders/", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/confirm-payment") {
			http.NotFound(w, r)
			return
		}
		orderID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/admin/orders/"), "/confirm-payment")
		if orderID == "" {
			writeError(w, http.StatusBadRequest, errString("missing order id"))
			return
		}
		result, err := svc.AdminConfirmOrderPayment(orderID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}))

	mux.HandleFunc("/api/admin/payments", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		payments, err := svc.ListPaymentsFiltered(
			strings.TrimSpace(r.URL.Query().Get("user_id")),
			strings.TrimSpace(r.URL.Query().Get("status")),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, payments)
	}))

	mux.HandleFunc("/api/admin/usage-logs", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		logs, err := svc.ListUsageLogsFiltered(
			strings.TrimSpace(r.URL.Query().Get("provider")),
			strings.TrimSpace(r.URL.Query().Get("alias")),
			strings.TrimSpace(r.URL.Query().Get("status")),
			strings.TrimSpace(r.URL.Query().Get("error_type")),
			strings.TrimSpace(r.URL.Query().Get("account_id")),
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, logs)
	}))

	mux.HandleFunc("/api/admin/operations/healthcheck", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := runHealthChecks(r.Context(), svc, providerRegistry); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"accounts": data.UpstreamAccounts})
	}))

	mux.HandleFunc("/api/admin/operations/refresh", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := refreshExpiringAccounts(r.Context(), svc, oauthHTTPClient, loadOAuthProviderConfigs(), automation); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"accounts": data.UpstreamAccounts})
	}))

	mux.HandleFunc("/api/admin/dispatch-debug", requireRole(svc, "admin", func(w http.ResponseWriter, r *http.Request, _ domain.User) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = refreshExpiringAccounts(r.Context(), svc, oauthHTTPClient, loadOAuthProviderConfigs(), automation)
		var req struct {
			APIKey     string `json:"api_key"`
			ModelAlias string `json:"model_alias"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		trace, err := service.ExplainDispatchInData(data, req.ModelAlias, req.APIKey)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, trace)
	}))

	mux.HandleFunc("/api/public/catalog", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"packages":                  data.ServicePackages,
			"model_alias_policies":      data.ModelAliasPolicies,
			"proxy_core_mode":           runtimeConfig.Mode,
			"proxy_core_primary":        runtimeConfig.Primary,
			"proxy_core_primary_origin": runtimeConfig.PrimaryOrigin,
			"proxy_core_origin":         runtimeConfig.PrimaryOrigin,
			"proxy_core_origins":        runtimeConfig.Origins,
			"oauth_providers":           mapsKeys(loadOAuthProviderConfigs()),
		})
	})

	mux.HandleFunc("/api/admin/oauth/callback/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		providerName := strings.TrimPrefix(r.URL.Path, "/api/admin/oauth/callback/")
		cfg, ok := loadOAuthProviderConfigs()[providerName]
		if !ok {
			writeError(w, http.StatusBadRequest, errString("unknown oauth provider"))
			return
		}
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if state == "" || code == "" {
			_ = svc.MarkOAuthSessionFailed(state, "missing oauth callback code or state")
			writeError(w, http.StatusBadRequest, errString("missing oauth callback code or state"))
			return
		}
		session, err := svc.OAuthSessionByState(state)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		tokenPayload, err := exchangeAuthorizationCode(r.Context(), oauthHTTPClient, cfg, session, code)
		if err != nil {
			_ = svc.MarkOAuthSessionFailed(state, err.Error())
			writeError(w, http.StatusBadRequest, err)
			return
		}
		account, completedSession, err := svc.CompleteOAuthSession(state, tokenPayload)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeHTML(w, http.StatusOK, renderOAuthCallbackPage(account, completedSession.RedirectTo))
	})

	mux.HandleFunc("/api/user/me", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, safeUser(user))
	}))

	mux.HandleFunc("/api/user/catalog", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		packages, err := svc.UserPackages(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"packages": packages})
	}))

	mux.HandleFunc("/api/user/subscriptions", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		subscriptions, err := svc.UserSubscriptions(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, subscriptions)
	}))

	mux.HandleFunc("/api/user/profile", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		profile, err := svc.UserProfile(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		profile.User = safeUser(profile.User)
		writeJSON(w, http.StatusOK, profile)
	}))

	mux.HandleFunc("/api/user/usage-logs", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		logs, err := svc.UserUsageLogs(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, logs)
	}))

	mux.HandleFunc("/api/user/api-keys", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		switch r.Method {
		case http.MethodGet:
			profile, err := svc.UserProfile(user.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, profile.APIKeys)
		case http.MethodPost:
			var req struct {
				PackageID string `json:"package_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			key, err := svc.CreateUserAPIKey(user.ID, req.PackageID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, key)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/api/user/api-keys/", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if !strings.HasSuffix(r.URL.Path, "/revoke") || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		keyID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/user/api-keys/"), "/revoke")
		key, err := svc.RevokeUserAPIKey(user.ID, keyID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, key)
	}))

	mux.HandleFunc("/api/user/orders", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		switch r.Method {
		case http.MethodGet:
			profile, err := svc.UserProfile(user.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, profile.Orders)
		case http.MethodPost:
			var req struct {
				PackageID    string `json:"package_id"`
				BindAPIKeyID string `json:"bind_api_key_id"`
				CreateAPIKey bool   `json:"create_api_key"`
				AutoRenew    bool   `json:"auto_renew"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			checkout, err := svc.CreateCheckoutOrder(user.ID, req.PackageID, req.BindAPIKeyID, req.CreateAPIKey, req.AutoRenew, publicOrigin)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, checkout)
		default:
			http.NotFound(w, r)
		}
	}))

	mux.HandleFunc("/api/user/orders/", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		path := strings.TrimPrefix(r.URL.Path, "/api/user/orders/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(path, "/confirm-payment") {
			if r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			orderID := strings.TrimSuffix(path, "/confirm-payment")
			checkout, err := svc.ConfirmUserOrderPayment(user.ID, orderID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, checkout)
			return
		}
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		detail, err := svc.UserOrderDetail(user.ID, path)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}))

	mux.HandleFunc("/api/user/payments", requireRole(svc, "", func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		profile, err := svc.UserProfile(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, profile.Payments)
	}))

	mux.HandleFunc("/api/public/payments/webhook", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req struct {
			PaymentID   string `json:"payment_id"`
			ProviderRef string `json:"provider_ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		checkout, err := svc.CompletePayment(req.PaymentID, req.ProviderRef)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, checkout)
	})

	mux.HandleFunc("/mockpay/checkout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		paymentID := strings.TrimSpace(r.URL.Query().Get("payment_id"))
		if paymentID == "" {
			writeHTML(w, http.StatusBadRequest, []byte("<html><body><p>missing payment_id</p></body></html>"))
			return
		}
		body := `<html><body style="font-family: sans-serif; padding: 24px;"><h1>MockPay Checkout</h1><p>payment_id: ` + paymentID + `</p><button onclick="fetch('/api/public/payments/webhook',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({payment_id:'` + paymentID + `',provider_ref:'mockpay-demo'})}).then(r=>r.json()).then(()=>location.href='/').catch(err=>alert(err))">Mark Paid</button></body></html>`
		writeHTML(w, http.StatusOK, []byte(body))
	})

	mux.HandleFunc("/api/internal/platform-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = refreshExpiringAccounts(r.Context(), svc, oauthHTTPClient, loadOAuthProviderConfigs(), automation)
		data, err := svc.Data()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, data)
	})

	mux.HandleFunc("/api/internal/validate-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		apiKey, pkg, user, err := svc.ValidateAPIKey(req.Key)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"api_key":     apiKey,
			"package":     pkg,
			"user_group":  user.Group,
			"rpm":         user.RPM,
			"tpm":         user.TPM,
			"total_quota": user.TotalQuota,
		})
	})

	mux.HandleFunc("/api/internal/usage-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var usageLog domain.UsageLog
		if err := json.NewDecoder(r.Body).Decode(&usageLog); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if usageLog.CreatedAt.IsZero() {
			usageLog.CreatedAt = time.Now().UTC()
		}
		if err := svc.AppendUsageLog(usageLog); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if usageLog.AccountID != "" {
			_, _ = svc.RecordUsageOutcome(usageLog.AccountID, usageLog.ErrorType, usageLog.ErrorMessage, healthPolicy)
		}
		writeJSON(w, http.StatusCreated, usageLog)
	})

	webRoot := filepath.Join("apps", "web")
	webFiles := http.FileServer(http.Dir(webRoot))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, http.StatusOK, map[string]any{
				"service": "control-plane",
				"message": "control-plane API is running",
				"docs": []string{
					"/healthz",
					"/api/public/catalog",
					"/api/admin/kernel-status",
				},
			})
			return
		}
		webFiles.ServeHTTP(w, r)
	}))

	log.Printf("control-plane listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func openConfiguredStoreWithRetry(ctx context.Context, backend, dataPath, databaseURL string) (store.Store, error) {
	if backend != "postgres" {
		return store.NewConfiguredStore(ctx, backend, dataPath, databaseURL)
	}

	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		st, err := store.NewConfiguredStore(ctx, backend, dataPath, databaseURL)
		if err == nil {
			if attempt > 1 {
				log.Printf("postgres store connected after %d attempts", attempt)
			}
			return st, nil
		}
		lastErr = err
		log.Printf("postgres store init attempt %d/30 failed: %v", attempt, err)
		time.Sleep(2 * time.Second)
	}

	return nil, lastErr
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func controlPlanePublicOrigin(addr string) string {
	if origin := strings.TrimSpace(os.Getenv("CONTROL_PLANE_PUBLIC_ORIGIN")); origin != "" {
		return strings.TrimRight(origin, "/")
	}
	return "http://127.0.0.1" + addr
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func errString(value string) error { return stringErr(value) }

func findAccount(data domain.PlatformData, id string) (domain.UpstreamAccount, error) {
	for _, account := range data.UpstreamAccounts {
		if account.ID == id {
			return account, nil
		}
	}
	return domain.UpstreamAccount{}, errors.New("account not found")
}
