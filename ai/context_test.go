package ai

import (
	"fmt"
	"testing"
	"time"

	"goChat/models"
	"gorm.io/gorm"
)

func TestCompressHistoryByDateDecaysOlderDays(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.Local)
	var history []models.AIMessage
	addTurns := func(dayAge, count int) {
		for i := 0; i < count; i++ {
			createdAt := now.AddDate(0, 0, -dayAge).Add(time.Duration(i) * time.Minute)
			history = append(history,
				testHistoryMessage("user", fmt.Sprintf("d%d-u%d", dayAge, i), createdAt),
				testHistoryMessage("assistant", fmt.Sprintf("d%d-a%d", dayAge, i), createdAt.Add(time.Second)),
			)
		}
	}
	addTurns(2, 6)
	addTurns(1, 6)
	addTurns(0, 6)

	compressed := CompressHistoryByDate(history, now, 100)
	counts := map[int]int{}
	for _, message := range compressed {
		counts[calendarDayAge(message.CreatedAt, now)]++
	}

	if counts[0] != 12 || counts[1] != 6 || counts[2] != 4 {
		t.Fatalf("unexpected retained message counts by day: %#v", counts)
	}
}

func TestCompressHistoryByDateKeepsTurnsAndRecentLimit(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.Local)
	var history []models.AIMessage
	for i := 0; i < 8; i++ {
		createdAt := now.Add(time.Duration(i) * time.Minute)
		history = append(history,
			testHistoryMessage("user", fmt.Sprintf("u%d", i), createdAt),
			testHistoryMessage("assistant", fmt.Sprintf("a%d", i), createdAt.Add(time.Second)),
		)
	}

	compressed := CompressHistoryByDate(history, now, 6)
	if len(compressed) != 6 {
		t.Fatalf("expected 6 messages, got %d", len(compressed))
	}
	for i := 0; i < len(compressed); i += 2 {
		if compressed[i].Role != "user" || compressed[i+1].Role != "assistant" {
			t.Fatalf("turn was split: %#v", compressed[i:i+2])
		}
	}
	if compressed[0].Content != "u5" || compressed[len(compressed)-1].Content != "a7" {
		t.Fatalf("expected the three newest turns, got %#v", compressed)
	}
}

func TestBuildHybridHistoryKeepsRecentWindowAndDecaysOlderTurns(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.Local)
	var history []models.AIMessage
	for dayAge := 2; dayAge >= 0; dayAge-- {
		for i := 0; i < 4; i++ {
			createdAt := now.AddDate(0, 0, -dayAge).Add(time.Duration(i) * time.Minute)
			history = append(history,
				testHistoryMessage("user", fmt.Sprintf("d%d-u%d", dayAge, i), createdAt),
				testHistoryMessage("assistant", fmt.Sprintf("d%d-a%d", dayAge, i), createdAt.Add(time.Second)),
			)
		}
	}

	hybrid := BuildHybridHistory(history, now, 12, 6)
	if len(hybrid) != 12 {
		t.Fatalf("expected history limit to be filled, got %d", len(hybrid))
	}
	if hybrid[len(hybrid)-6].Content != "d0-u1" || hybrid[len(hybrid)-1].Content != "d0-a3" {
		t.Fatalf("recent sliding window was not preserved: %#v", hybrid)
	}
	for i := 0; i < len(hybrid); i += 2 {
		if hybrid[i].Role != "user" || hybrid[i+1].Role != "assistant" {
			t.Fatalf("hybrid context split a turn: %#v", hybrid[i:i+2])
		}
	}
}

func testHistoryMessage(role, content string, createdAt time.Time) models.AIMessage {
	return models.AIMessage{
		Model:   gorm.Model{CreatedAt: createdAt},
		Role:    role,
		Content: content,
		Status:  models.AIMessageCompleted,
	}
}
