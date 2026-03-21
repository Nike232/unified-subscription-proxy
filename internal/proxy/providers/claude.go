package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

const anthropicVersion = "2023-06-01"

type ClaudeProvider struct {
	client *http.Client
}

func NewClaudeProvider(client *http.Client) *ClaudeProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &ClaudeProvider{client: client}
}

func (p *ClaudeProvider) Name() string {
	return domain.ProviderClaude
}

func (p *ClaudeProvider) Execute(ctx context.Context, req Request) (Response, error) {
	baseURL := strings.TrimRight(firstNonEmpty(req.Account.Meta["base_url"], "https://api.anthropic.com"), "/")
	body, err := convertOpenAIChatToClaude(req.UpstreamModel, req.Body)
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	applyClaudeAuthHeaders(httpReq, req.Account)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	stream, _ := detectStream(req.Body)
	if stream && resp.StatusCode < 400 {
		reader, err := convertClaudeStreamToOpenAI(req.UpstreamModel, resp.Body)
		if err != nil {
			resp.Body.Close()
			return Response{}, err
		}
		return Response{
			StatusCode: resp.StatusCode,
			Headers:    http.Header{"Content-Type": []string{"text/event-stream; charset=utf-8"}},
			Stream:     reader,
		}, nil
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode >= 400 {
		return Response{
			StatusCode: resp.StatusCode,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       convertClaudeError(resp.StatusCode, raw),
		}, nil
	}
	out, err := convertClaudeToOpenAI(req.UpstreamModel, raw)
	if err != nil {
		return Response{}, err
	}
	return Response{
		StatusCode: resp.StatusCode,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       out,
	}, nil
}

func (p *ClaudeProvider) HealthCheck(ctx context.Context, account domain.UpstreamAccount) (HealthResult, error) {
	baseURL := strings.TrimRight(firstNonEmpty(account.Meta["base_url"], "https://api.anthropic.com"), "/")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return HealthResult{}, err
	}
	applyClaudeAuthHeaders(httpReq, account)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return HealthResult{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	result := HealthResult{
		OK:         resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
	}
	if result.OK {
		result.Message = "claude account reachable"
	} else {
		result.Message = string(body)
	}
	return result, nil
}

type claudeMessageRequest struct {
	Model       string                     `json:"model"`
	System      string                     `json:"system,omitempty"`
	Messages    []claudeMessage            `json:"messages"`
	MaxTokens   int                        `json:"max_tokens"`
	Temperature any                        `json:"temperature,omitempty"`
	TopP        any                        `json:"top_p,omitempty"`
	Stream      bool                       `json:"stream,omitempty"`
	Metadata    map[string]any             `json:"metadata,omitempty"`
	Extra       map[string]json.RawMessage `json:"-"`
}

type claudeMessage struct {
	Role    string       `json:"role"`
	Content []claudePart `json:"content"`
}

type claudePart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func convertOpenAIChatToClaude(upstreamModel string, raw []byte) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	messagesRaw, _ := payload["messages"].([]any)
	req := claudeMessageRequest{
		Model:     firstNonEmpty(upstreamModel, stringValue(payload["model"])),
		MaxTokens: intValue(payload["max_tokens"], 1024),
		Stream:    boolValue(payload["stream"]),
	}
	if value, ok := payload["temperature"]; ok {
		req.Temperature = value
	}
	if value, ok := payload["top_p"]; ok {
		req.TopP = value
	}
	for _, item := range messagesRaw {
		msg, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text := flattenContent(msg["content"])
		role := stringValue(msg["role"])
		if role == "system" {
			if req.System == "" {
				req.System = text
			} else {
				req.System += "\n" + text
			}
			continue
		}
		if role == "" {
			role = "user"
		}
		req.Messages = append(req.Messages, claudeMessage{
			Role: role,
			Content: []claudePart{{
				Type: "text",
				Text: text,
			}},
		})
	}
	if len(req.Messages) == 0 {
		req.Messages = []claudeMessage{{Role: "user", Content: []claudePart{{Type: "text", Text: ""}}}}
	}
	return json.Marshal(req)
}

func convertClaudeToOpenAI(model string, raw []byte) ([]byte, error) {
	var resp claudeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	var content strings.Builder
	for _, part := range resp.Content {
		if part.Type == "text" {
			content.WriteString(part.Text)
		}
	}
	openAIResp := map[string]any{
		"id":     firstNonEmpty(resp.ID, "chatcmpl_claude_proxy"),
		"object": "chat.completion",
		"model":  model,
		"choices": []map[string]any{{
			"index":         0,
			"finish_reason": claudeStopReason(resp.StopReason),
			"message": map[string]any{
				"role":    "assistant",
				"content": content.String(),
			},
		}},
		"usage": map[string]any{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
	return json.Marshal(openAIResp)
}

func convertClaudeError(statusCode int, raw []byte) []byte {
	type claudeErr struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	parsed := claudeErr{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return []byte(`{"error":{"message":"claude upstream error","type":"upstream_error"}}`)
	}
	errorType := "upstream_error"
	switch parsed.Error.Type {
	case "authentication_error", "permission_error":
		errorType = "authentication_error"
	case "rate_limit_error", "overloaded_error":
		errorType = "insufficient_quota"
	case "not_found_error":
		errorType = "model_not_found"
	case "invalid_request_error":
		errorType = "invalid_request_error"
	default:
		if statusCode >= 500 {
			errorType = "upstream_error"
		}
	}
	out, err := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": firstNonEmpty(parsed.Error.Message, "claude upstream error"),
			"type":    errorType,
		},
	})
	if err != nil {
		return []byte(`{"error":{"message":"claude upstream error","type":"upstream_error"}}`)
	}
	return out
}

func convertClaudeStreamToOpenAI(model string, source io.ReadCloser) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		defer source.Close()
		defer pw.Close()

		scanner := bufio.NewScanner(source)
		scanner.Buffer(make([]byte, 1024), 1024*1024)
		var eventType string
		firstChunk := true
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload == "" || payload == "[DONE]" {
					continue
				}
				switch eventType {
				case "content_block_delta":
					var event struct {
						Delta struct {
							Text string `json:"text"`
						} `json:"delta"`
					}
					if json.Unmarshal([]byte(payload), &event) == nil && event.Delta.Text != "" {
						chunk := map[string]any{
							"id":     "chatcmpl_claude_stream_proxy",
							"object": "chat.completion.chunk",
							"model":  model,
							"choices": []map[string]any{{
								"index": 0,
								"delta": map[string]any{
									"content": event.Delta.Text,
								},
								"finish_reason": nil,
							}},
						}
						if firstChunk {
							chunk["choices"].([]map[string]any)[0]["delta"].(map[string]any)["role"] = "assistant"
							firstChunk = false
						}
						if err := writeSSEChunk(pw, chunk); err != nil {
							_ = pw.CloseWithError(err)
							return
						}
					}
				case "message_stop":
					finish := map[string]any{
						"id":     "chatcmpl_claude_stream_proxy",
						"object": "chat.completion.chunk",
						"model":  model,
						"choices": []map[string]any{{
							"index":         0,
							"delta":         map[string]any{},
							"finish_reason": "stop",
						}},
					}
					if err := writeSSEChunk(pw, finish); err != nil {
						_ = pw.CloseWithError(err)
						return
					}
					if _, err := io.WriteString(pw, "data: [DONE]\n\n"); err != nil {
						_ = pw.CloseWithError(err)
						return
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			_ = pw.CloseWithError(err)
		}
	}()
	return pr, nil
}

func applyClaudeAuthHeaders(req *http.Request, account domain.UpstreamAccount) {
	if apiKey := strings.TrimSpace(account.Meta["api_key"]); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
		return
	}
	if accessToken := strings.TrimSpace(account.Meta["access_token"]); accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
}

func claudeStopReason(value string) string {
	switch value {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return "stop"
	}
}
