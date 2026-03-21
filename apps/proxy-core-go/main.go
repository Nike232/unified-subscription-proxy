package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"unifiedsubscriptionproxy/internal/platform/client"
	"unifiedsubscriptionproxy/internal/platform/domain"
	"unifiedsubscriptionproxy/internal/platform/service"
	proxyproviders "unifiedsubscriptionproxy/internal/proxy/providers"
)

type dispatchRequest struct {
	ModelAlias string `json:"model_alias"`
	Input      string `json:"input"`
}

type usageLogAppender interface {
	AppendUsageLog(context.Context, domain.UsageLog) error
}

type providerResolver func(string) (proxyproviders.Provider, error)

func main() {
	addr := getenv("PROXY_CORE_ADDR", ":8081")
	controlPlaneOrigin := getenv("CONTROL_PLANE_ORIGIN", "http://127.0.0.1:8080")
	cpClient := client.NewControlPlaneClient(controlPlaneOrigin)
	providerRegistry := proxyproviders.NewRegistry(http.DefaultClient)

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":               "ok",
			"service":              "proxy-core",
			"control_plane_origin": controlPlaneOrigin,
		})
	})

	mux.HandleFunc("/api/v1/models", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		data, err := cpClient.Snapshot(r.Context())
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
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

		payload := map[string]any{
			"model": req.ModelAlias,
			"messages": []map[string]any{
				{"role": "user", "content": req.Input},
			},
		}
		rawBody, err := json.Marshal(payload)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		upstream, result, usageLog, err := executeChatCompletion(r.Context(), cpClient, providerRegistry, apiKey, rawBody)
		if err != nil {
			writeError(w, statusCodeForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"dispatch": result,
			"usage": map[string]any{
				"log_id":         usageLog.ID,
				"api_key_id":     usageLog.APIKeyID,
				"provider":       usageLog.Provider,
				"upstream_model": usageLog.UpstreamModel,
			},
			"upstream_status": upstream.StatusCode,
			"upstream_body":   json.RawMessage(upstream.Body),
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

		rawBody, err := readRawBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		upstream, _, _, err := executeChatCompletion(r.Context(), cpClient, providerRegistry, apiKey, rawBody)
		if err != nil {
			writeError(w, statusCodeForError(err), err)
			return
		}
		copyResponse(w, upstream)
	})

	log.Printf("proxy-core listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func executeChatCompletion(
	ctx context.Context,
	cpClient *client.ControlPlaneClient,
	providerRegistry *proxyproviders.Registry,
	apiKey string,
	rawBody []byte,
) (proxyproviders.Response, service.DispatchResult, domain.UsageLog, error) {
	data, err := cpClient.Snapshot(ctx)
	if err != nil {
		return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, err
	}
	modelAlias, err := extractModelAlias(rawBody)
	if err != nil {
		return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, err
	}
	trace, err := service.ExplainDispatchInData(data, modelAlias, apiKey)
	if err != nil {
		return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, err
	}
	stream, _ := detectStream(rawBody)
	return executeAgainstTrace(ctx, cpClient, providerRegistry.Provider, data, trace, rawBody, stream)
}

func executeAgainstTrace(
	ctx context.Context,
	logAppender usageLogAppender,
	providerFor providerResolver,
	data domain.PlatformData,
	trace service.DispatchTrace,
	rawBody []byte,
	stream bool,
) (proxyproviders.Response, service.DispatchResult, domain.UsageLog, error) {
	var lastProviderErr error
	for _, candidate := range trace.Candidates {
		result, usageLog := buildDispatchArtifacts(trace, candidate, stream)
		account, err := findAccount(data, candidate.AccountID)
		if err != nil {
			return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, err
		}
		provider, err := providerFor(candidate.Provider)
		if err != nil {
			return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, err
		}
		resp, err := provider.Execute(ctx, proxyproviders.Request{
			ModelAlias:    result.ModelAlias,
			UpstreamModel: result.UpstreamModel,
			Account:       account,
			Headers:       http.Header{},
			Body:          rawBody,
		})
		if err != nil {
			usageLog.Status = "provider_error"
			usageLog.ErrorType = "provider_error"
			usageLog.ErrorMessage = err.Error()
			_ = logAppender.AppendUsageLog(ctx, usageLog)
			lastProviderErr = err
			continue
		}

		usageLog.UpstreamStatusCode = resp.StatusCode
		if resp.StatusCode < 400 {
			usageLog.Status = "completed"
			if err := logAppender.AppendUsageLog(ctx, usageLog); err != nil {
				return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, err
			}
			return resp, result, usageLog, nil
		}

		usageLog.Status = "upstream_error"
		usageLog.ErrorType, usageLog.ErrorMessage = classifyUpstreamError(resp)
		_ = logAppender.AppendUsageLog(ctx, usageLog)
		if shouldFallback(usageLog.ErrorType) {
			continue
		}
		return resp, result, usageLog, nil
	}
	if lastProviderErr != nil {
		return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, lastProviderErr
	}
	return proxyproviders.Response{}, service.DispatchResult{}, domain.UsageLog{}, errors.New("no provider candidate succeeded")
}

func buildDispatchArtifacts(trace service.DispatchTrace, candidate service.Candidate, stream bool) (service.DispatchResult, domain.UsageLog) {
	result := service.DispatchResult{
		APIKeyID:       trace.APIKey.ID,
		PackageID:      trace.APIKey.PackageID,
		ModelAlias:     trace.ModelAlias,
		Provider:       candidate.Provider,
		AccountID:      candidate.AccountID,
		UpstreamModel:  candidate.UpstreamModel,
		CandidateCount: len(trace.Candidates),
	}
	usageLog := domain.UsageLog{
		ID:            "log-" + randomID(6),
		APIKeyID:      trace.APIKey.ID,
		UserID:        trace.APIKey.UserID,
		ModelAlias:    trace.ModelAlias,
		Provider:      candidate.Provider,
		AccountID:     candidate.AccountID,
		UpstreamModel: candidate.UpstreamModel,
		Status:        "dispatched",
		RequestKind:   "chat_completions",
		CreatedAt:     timeNowUTC(),
	}
	if stream {
		usageLog.RequestKind = "chat_completions_stream"
	}
	return result, usageLog
}

func findAccount(data domain.PlatformData, id string) (domain.UpstreamAccount, error) {
	for _, account := range data.UpstreamAccounts {
		if account.ID == id {
			return account, nil
		}
	}
	return domain.UpstreamAccount{}, errors.New("selected account not found in platform snapshot")
}

func extractModelAlias(rawBody []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return "", err
	}
	modelAlias, _ := payload["model"].(string)
	if strings.TrimSpace(modelAlias) == "" {
		return "", errors.New("model is required")
	}
	return modelAlias, nil
}

func readRawBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func copyResponse(w http.ResponseWriter, resp proxyproviders.Response) {
	for key, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	if resp.Stream != nil {
		defer resp.Stream.Close()
		_, _ = io.Copy(w, resp.Stream)
		return
	}
	_, _ = w.Write(resp.Body)
}

func detectStream(rawBody []byte) (bool, error) {
	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return false, err
	}
	stream, _ := payload["stream"].(bool)
	return stream, nil
}

func classifyUpstreamError(resp proxyproviders.Response) (string, string) {
	message := "upstream request failed"
	type payload struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	var parsed payload
	if len(resp.Body) > 0 && json.Unmarshal(resp.Body, &parsed) == nil {
		if strings.TrimSpace(parsed.Error.Message) != "" {
			message = parsed.Error.Message
		}
		switch parsed.Error.Type {
		case "insufficient_quota":
			return "quota_exceeded", message
		case "invalid_request_error":
			return "invalid_request", message
		case "authentication_error":
			return "auth_failed", message
		case "model_not_found":
			return "model_unavailable", message
		case "upstream_error":
			return upstreamErrorType(resp.StatusCode), message
		}
	}
	return upstreamErrorType(resp.StatusCode), message
}

func shouldFallback(errorType string) bool {
	switch errorType {
	case "auth_failed", "quota_exceeded", "model_unavailable", "upstream_unavailable", "provider_error":
		return true
	default:
		return false
	}
}

func upstreamErrorType(statusCode int) string {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return "auth_failed"
	case statusCode == http.StatusTooManyRequests:
		return "quota_exceeded"
	case statusCode >= 500:
		return "upstream_unavailable"
	default:
		return "upstream_error"
	}
}

func statusCodeForError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case strings.Contains(err.Error(), "invalid api key"):
		return http.StatusUnauthorized
	case strings.Contains(err.Error(), "model alias not found"):
		return http.StatusBadRequest
	case strings.Contains(err.Error(), "no available upstream account"):
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadGateway
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

var timeNowUTC = func() time.Time {
	return time.Now().UTC()
}

func randomID(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf)
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func errString(value string) error { return stringErr(value) }
