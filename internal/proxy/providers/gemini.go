package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type GeminiProvider struct {
	client *http.Client
}

func NewGeminiProvider(client *http.Client) *GeminiProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &GeminiProvider{client: client}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Execute(ctx context.Context, req Request) (Response, error) {
	apiKey := strings.TrimSpace(req.Account.Meta["api_key"])
	if apiKey == "" {
		return Response{}, fmt.Errorf("gemini provider missing api_key for account %s", req.Account.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(req.Account.Meta["base_url"]), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	geminiBody, err := convertOpenAIChatToGemini(req.UpstreamModel, req.Body)
	if err != nil {
		return Response{}, err
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, req.UpstreamModel, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(geminiBody))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	stream, _ := detectStream(req.Body)
	if resp.StatusCode >= 400 {
		return Response{
			StatusCode: resp.StatusCode,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       convertGeminiError(body),
		}, nil
	}

	openAIBody, headers, err := convertGeminiToOpenAI(req.UpstreamModel, body, stream)
	if err != nil {
		return Response{}, err
	}
	return Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       openAIBody,
	}, nil
}

func (p *GeminiProvider) HealthCheck(ctx context.Context, account domain.UpstreamAccount) (HealthResult, error) {
	apiKey := strings.TrimSpace(account.Meta["api_key"])
	if apiKey == "" {
		return HealthResult{}, fmt.Errorf("gemini provider missing api_key for account %s", account.ID)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(account.Meta["base_url"]), "/")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	url := fmt.Sprintf("%s/v1beta/models?key=%s", baseURL, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return HealthResult{}, err
	}
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
		result.Message = "gemini account reachable"
	} else {
		result.Message = string(body)
	}
	return result, nil
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type geminiGenerateContentRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  map[string]any  `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata map[string]any `json:"usageMetadata"`
}

func convertOpenAIChatToGemini(model string, raw []byte) ([]byte, error) {
	var req openAIChatRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}

	out := geminiGenerateContentRequest{}
	out.GenerationConfig = map[string]any{}
	for _, message := range req.Messages {
		text := flattenContent(message.Content)
		switch message.Role {
		case "system":
			out.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: text}}}
		case "assistant":
			out.Contents = append(out.Contents, geminiContent{Role: "model", Parts: []geminiPart{{Text: text}}})
		default:
			out.Contents = append(out.Contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: text}}})
		}
	}
	if len(out.Contents) == 0 {
		out.Contents = []geminiContent{{Role: "user", Parts: []geminiPart{{Text: ""}}}}
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		for _, key := range []string{"temperature", "top_p", "max_tokens"} {
			if value, ok := payload[key]; ok {
				switch key {
				case "max_tokens":
					out.GenerationConfig["maxOutputTokens"] = value
				case "top_p":
					out.GenerationConfig["topP"] = value
				default:
					out.GenerationConfig[key] = value
				}
			}
		}
		if len(out.GenerationConfig) == 0 {
			out.GenerationConfig = nil
		}
	}
	return json.Marshal(out)
}

func convertGeminiToOpenAI(model string, raw []byte, stream bool) ([]byte, http.Header, error) {
	var resp geminiGenerateContentResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, nil, err
	}
	content := ""
	if len(resp.Candidates) > 0 {
		for _, part := range resp.Candidates[0].Content.Parts {
			content += part.Text
		}
	}
	openAIResp := map[string]any{
		"id":      "chatcmpl_gemini_proxy",
		"object":  "chat.completion",
		"model":   model,
		"choices": []map[string]any{{"index": 0, "finish_reason": "stop", "message": map[string]any{"role": "assistant", "content": content}}},
	}
	if len(resp.UsageMetadata) > 0 {
		openAIResp["usage"] = map[string]any{
			"prompt_tokens":     resp.UsageMetadata["promptTokenCount"],
			"completion_tokens": resp.UsageMetadata["candidatesTokenCount"],
			"total_tokens":      resp.UsageMetadata["totalTokenCount"],
		}
	}
	if stream {
		streamPayload, err := buildGeminiStreamResponse(model, content)
		return streamPayload, http.Header{"Content-Type": []string{"text/event-stream; charset=utf-8"}}, err
	}
	body, err := json.Marshal(openAIResp)
	return body, http.Header{"Content-Type": []string{"application/json"}}, err
}

func convertGeminiError(raw []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return []byte(`{"error":{"message":"gemini upstream error","type":"upstream_error"}}`)
	}
	message := "gemini upstream error"
	if errPayload, ok := payload["error"].(map[string]any); ok {
		if msg, ok := errPayload["message"].(string); ok && strings.TrimSpace(msg) != "" {
			message = msg
		}
	}
	out, err := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "upstream_error",
		},
	})
	if err != nil {
		return []byte(`{"error":{"message":"gemini upstream error","type":"upstream_error"}}`)
	}
	return out
}

func flattenContent(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		var builder strings.Builder
		for _, item := range value {
			if part, ok := item.(map[string]any); ok {
				if text, ok := part["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		return builder.String()
	default:
		return ""
	}
}

func buildGeminiStreamResponse(model, content string) ([]byte, error) {
	chunk := map[string]any{
		"id":     "chatcmpl_gemini_stream_proxy",
		"object": "chat.completion.chunk",
		"model":  model,
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": nil,
			},
		},
	}
	finish := map[string]any{
		"id":     "chatcmpl_gemini_stream_proxy",
		"object": "chat.completion.chunk",
		"model":  model,
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": "stop",
			},
		},
	}
	chunkBody, err := json.Marshal(chunk)
	if err != nil {
		return nil, err
	}
	finishBody, err := json.Marshal(finish)
	if err != nil {
		return nil, err
	}
	payload := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: [DONE]\n\n", chunkBody, finishBody)
	return []byte(payload), nil
}

func detectStream(raw []byte) (bool, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, err
	}
	stream, _ := payload["stream"].(bool)
	return stream, nil
}
