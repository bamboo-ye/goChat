package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatibleProvider struct {
	name       string
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAICompatibleProvider(name, baseURL, apiKey, model string, timeout time.Duration) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *OpenAICompatibleProvider) Name() string {
	return p.name
}

type chatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []PromptMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type chatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message PromptMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAICompatibleProvider) Complete(ctx context.Context, request CompletionRequest) (CompletionResult, error) {
	if p.baseURL == "" || p.model == "" {
		return CompletionResult{}, fmt.Errorf("ai provider is not configured")
	}

	payload, err := json.Marshal(chatCompletionRequest{
		Model:       p.model,
		Messages:    request.Messages,
		Temperature: request.Temperature,
		MaxTokens:   request.MaxTokens,
	})
	if err != nil {
		return CompletionResult{}, fmt.Errorf("encode completion request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return CompletionResult{}, fmt.Errorf("create completion request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	response, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return CompletionResult{}, fmt.Errorf("call ai provider: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return CompletionResult{}, fmt.Errorf("read ai provider response: %w", err)
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return CompletionResult{}, fmt.Errorf("decode ai provider response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if completion.Error != nil && completion.Error.Message != "" {
			return CompletionResult{}, fmt.Errorf("ai provider error: %s", completion.Error.Message)
		}
		return CompletionResult{}, fmt.Errorf("ai provider returned status %d", response.StatusCode)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return CompletionResult{}, fmt.Errorf("ai provider returned an empty response")
	}

	return CompletionResult{
		Content:          strings.TrimSpace(completion.Choices[0].Message.Content),
		Model:            completion.Model,
		PromptTokens:     completion.Usage.PromptTokens,
		CompletionTokens: completion.Usage.CompletionTokens,
	}, nil
}
