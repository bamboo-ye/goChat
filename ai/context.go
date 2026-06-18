package ai

import (
	"sort"
	"time"

	"goChat/models"
)

type conversationTurn struct {
	messages []models.AIMessage
	dayAge   int
}

func BuildHybridHistory(history []models.AIMessage, now time.Time, messageLimit, recentMessageLimit int) []models.AIMessage {
	if messageLimit <= 0 || len(history) == 0 {
		return nil
	}
	turns := buildConversationTurns(history, now)
	if len(turns) == 0 {
		return nil
	}

	recentBudget := recentMessageLimit
	if recentBudget <= 0 || recentBudget > messageLimit {
		recentBudget = messageLimit
	}
	recentStart := len(turns)
	recentUsed := 0
	for i := len(turns) - 1; i >= 0; i-- {
		size := len(turns[i].messages)
		if recentUsed+size > recentBudget {
			break
		}
		recentStart = i
		recentUsed += size
	}

	recent := flattenTurns(turns[recentStart:])
	remaining := messageLimit - len(recent)
	if remaining <= 0 || recentStart == 0 {
		return recent
	}
	older := CompressHistoryByDate(flattenTurns(turns[:recentStart]), now, remaining)
	return append(older, recent...)
}

// CompressHistoryByDate keeps recent conversations dense while progressively
// sampling older days. Messages are retained as user/assistant turns so the
// model does not receive a question without its corresponding answer.
func CompressHistoryByDate(history []models.AIMessage, now time.Time, messageLimit int) []models.AIMessage {
	if messageLimit <= 0 || len(history) == 0 {
		return nil
	}

	turns := buildConversationTurns(history, now)
	if len(turns) == 0 {
		return nil
	}

	byDay := make(map[int][]conversationTurn)
	var ages []int
	for _, turn := range turns {
		if _, exists := byDay[turn.dayAge]; !exists {
			ages = append(ages, turn.dayAge)
		}
		byDay[turn.dayAge] = append(byDay[turn.dayAge], turn)
	}

	var sampled []conversationTurn
	for _, age := range ages {
		dayTurns := byDay[age]
		keep := (len(dayTurns) + age) / (age + 1)
		if keep < 1 {
			keep = 1
		}
		sampled = append(sampled, dayTurns[len(dayTurns)-keep:]...)
	}
	sort.SliceStable(sampled, func(i, j int) bool {
		return sampled[i].messages[0].CreatedAt.Before(sampled[j].messages[0].CreatedAt)
	})

	selected := make([]conversationTurn, 0, len(sampled))
	used := 0
	for i := len(sampled) - 1; i >= 0; i-- {
		size := len(sampled[i].messages)
		if used+size > messageLimit {
			continue
		}
		selected = append(selected, sampled[i])
		used += size
	}

	result := make([]models.AIMessage, 0, used)
	for i := len(selected) - 1; i >= 0; i-- {
		result = append(result, selected[i].messages...)
	}
	return result
}

func buildConversationTurns(history []models.AIMessage, now time.Time) []conversationTurn {
	var turns []conversationTurn
	for _, message := range history {
		if message.Status != models.AIMessageCompleted {
			continue
		}
		switch message.Role {
		case "user":
			turns = append(turns, conversationTurn{
				messages: []models.AIMessage{message},
				dayAge:   calendarDayAge(message.CreatedAt, now),
			})
		case "assistant":
			if len(turns) == 0 || len(turns[len(turns)-1].messages) > 1 {
				turns = append(turns, conversationTurn{
					messages: []models.AIMessage{message},
					dayAge:   calendarDayAge(message.CreatedAt, now),
				})
				continue
			}
			turns[len(turns)-1].messages = append(turns[len(turns)-1].messages, message)
		}
	}
	return turns
}

func flattenTurns(turns []conversationTurn) []models.AIMessage {
	var messages []models.AIMessage
	for _, turn := range turns {
		messages = append(messages, turn.messages...)
	}
	return messages
}

func calendarDayAge(createdAt, now time.Time) int {
	if createdAt.IsZero() {
		return 0
	}
	location := now.Location()
	created := createdAt.In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	createdDay := time.Date(created.Year(), created.Month(), created.Day(), 0, 0, 0, 0, time.UTC)
	age := int(today.Sub(createdDay).Hours() / 24)
	if age < 0 {
		return 0
	}
	return age
}
