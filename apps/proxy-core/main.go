package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/service"
	"unifiedsubscriptionproxy/internal/platform/store"
)

type dispatchRequest struct {
	ModelAlias string `json:"model_alias"`
	Input      string `json:"input"`
}

func main() {
	addr := getenv("PROXY_CORE_ADDR", ":8081")
	dataPath := getenv("PLATFORM_DATA_FILE", "./data/platform.json")
	svc := service.New(store.NewFileStore(dataPath))

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "proxy-core"})
	})

	mux.HandleFunc("/api/v1/models", func(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusOK, data.ModelAliasPolicies)
	})

	mux.HandleFunc("/api/v1/dispatch", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		apiKey := bearerToken(r.Header.Get("Authorization"))
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, errString("missing bearer api key"))
			return
		}

		var req dispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result, err := svc.ResolveDispatch(req.ModelAlias, apiKey)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"dispatch": result,
			"message":  "mock upstream dispatch completed",
			"output":   "This is a simulated response from the unified proxy core.",
			"input":    req.Input,
		})
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		apiKey := bearerToken(r.Header.Get("Authorization"))
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, errString("missing bearer api key"))
			return
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		modelAlias, _ := payload["model"].(string)
		result, err := svc.ResolveDispatch(modelAlias, apiKey)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":       "chatcmpl_mock",
			"object":   "chat.completion",
			"model":    modelAlias,
			"provider": result.Provider,
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Unified proxy mock response",
					},
				},
			},
			"dispatch": result,
		})
	})

	log.Printf("proxy-core listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func bearerToken(value string) string {
	parts := strings.SplitN(strings.TrimSpace(value), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
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
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func errString(value string) error { return stringErr(value) }
