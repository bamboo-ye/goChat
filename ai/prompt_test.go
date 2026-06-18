package ai

import (
	"strings"
	"testing"

	"goChat/models"
)

func TestBuildPromptFiltersIncompleteMessages(t *testing.T) {
	prompt := BuildPrompt(CharacterSnapshot{Name: "小雨", RelationType: "bestie"}, []models.AIMessage{
		{Role: "user", Content: "你好", Status: models.AIMessageCompleted},
		{Role: "assistant", Content: "生成中", Status: models.AIMessagePending},
	})
	if len(prompt) != 2 {
		t.Fatalf("expected system and one completed message, got %d", len(prompt))
	}
	if !strings.Contains(prompt[0].Content, "小雨") || prompt[1].Content != "你好" {
		t.Fatalf("unexpected prompt: %#v", prompt)
	}
	for _, instruction := range []string{"日常聊天", "回复应简短", "拆分成多个简短消息"} {
		if !strings.Contains(prompt[0].Content, instruction) {
			t.Fatalf("system prompt missing %q: %s", instruction, prompt[0].Content)
		}
	}
}
