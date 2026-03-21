package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/service"
	"unifiedsubscriptionproxy/internal/platform/store"
)

func main() {
	addr := getenv("CONTROL_PLANE_ADDR", ":8080")
	dataPath := getenv("PLATFORM_DATA_FILE", "./data/platform.json")
	proxyOrigin := getenv("PROXY_CORE_ORIGIN", "http://127.0.0.1:8081")

	svc := service.New(store.NewFileStore(dataPath))
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "control-plane"})
	})

	mux.HandleFunc("/api/admin/overview", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
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
	})

	mux.HandleFunc("/api/admin/data", func(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusOK, data)
	})

	mux.HandleFunc("/api/admin/upstream-accounts", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
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
	})

	mux.HandleFunc("/api/admin/upstream-accounts/", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/admin/upstream-accounts/")
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
	})

	mux.HandleFunc("/api/admin/packages", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
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
	})

	mux.HandleFunc("/api/admin/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
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
	})

	mux.HandleFunc("/api/admin/api-keys", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
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
	})

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
			"packages":             data.ServicePackages,
			"model_alias_policies": data.ModelAliasPolicies,
			"proxy_core_origin":    proxyOrigin,
		})
	})

	webRoot := filepath.Join("apps", "web")
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))

	log.Printf("control-plane listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func errString(value string) error { return stringErr(value) }
