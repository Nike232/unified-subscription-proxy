package providers

import (
	"context"
	"io"
	"net/http"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type Request struct {
	ModelAlias    string
	UpstreamModel string
	Account       domain.UpstreamAccount
	Headers       http.Header
	Body          []byte
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Stream     io.ReadCloser
}

type HealthResult struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type Provider interface {
	Name() string
	Execute(ctx context.Context, req Request) (Response, error)
	HealthCheck(ctx context.Context, account domain.UpstreamAccount) (HealthResult, error)
}
