package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAICompatibleProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatal("missing bearer token")
		}
		if request.Header.Get("HTTP-Referer") != "https://example.com" {
			t.Fatal("missing referer header")
		}
		if request.Header.Get("X-Title") != "goChat" {
			t.Fatal("missing title header")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"model":"test-model","choices":[{"message":{"role":"assistant","content":"你好"}}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProviderWithAttribution("test", server.URL, "secret", "test-model", "https://example.com", "goChat", time.Second)
	result, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []PromptMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "你好" || result.PromptTokens != 12 || result.CompletionTokens != 3 {
		t.Fatalf("unexpected completion: %#v", result)
	}
}

func TestOpenRouterProviderRequiresAPIKey(t *testing.T) {
	provider := NewOpenRouterProvider("https://openrouter.ai/api/v1", "", "openai/gpt-oss-120b:free", "", "goChat", time.Second)
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []PromptMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected missing configuration error")
	}
}
