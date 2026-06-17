package ai

import "context"

type PromptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Messages    []PromptMessage
	Temperature float64
	MaxTokens   int
}

type CompletionResult struct {
	Content          string
	Model            string
	PromptTokens     int
	CompletionTokens int
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, request CompletionRequest) (CompletionResult, error)
}
