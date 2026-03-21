package providers

import (
	"fmt"
	"net/http"
	"strings"
)

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(client *http.Client) *Registry {
	return &Registry{
		providers: map[string]Provider{
			"antigravity": NewAntigravityProvider(client),
			"claude":      NewClaudeProvider(client),
			"codex":       NewCodexProvider(client),
			"openai":      NewOpenAIProvider(client),
			"gemini":      NewGeminiProvider(client),
		},
	}
}

func (r *Registry) Provider(name string) (Provider, error) {
	provider, ok := r.providers[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("provider not implemented: %s", name)
	}
	return provider, nil
}
