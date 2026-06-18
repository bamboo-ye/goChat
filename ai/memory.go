package ai

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"goChat/models"
	"goChat/utils"
	"gorm.io/gorm/clause"
)

type MemoryConfig struct {
	Enabled            bool
	RecentMessageLimit int
	TopK               int
	CandidateLimit     int
	MinScore           float64
	VectorDimensions   int
	BackfillLimit      int
}

type RetrievedMemory struct {
	Content    string
	OccurredAt time.Time
	Score      float64
}

type scoredMemory struct {
	RetrievedMemory
	id uint
}

func normalizeMemoryConfig(config MemoryConfig) MemoryConfig {
	if config.RecentMessageLimit <= 0 {
		config.RecentMessageLimit = 8
	}
	if config.TopK <= 0 {
		config.TopK = 4
	}
	if config.CandidateLimit <= 0 {
		config.CandidateLimit = 200
	}
	if config.VectorDimensions <= 0 {
		config.VectorDimensions = 256
	}
	if config.BackfillLimit <= 0 {
		config.BackfillLimit = 400
	}
	return config
}

func (s *Service) retrieveLongTermMemories(ownerID, conversationID uint, query string, history []models.AIMessage, beforeMessageID uint) []RetrievedMemory {
	if !s.memory.Enabled || strings.TrimSpace(query) == "" {
		return nil
	}
	if err := s.syncLongTermMemories(ownerID, conversationID, beforeMessageID); err != nil {
		log.Printf("sync ai memories: %v", err)
	}

	candidateLimit := s.memory.CandidateLimit
	if candidateLimit <= 0 {
		candidateLimit = 200
	}
	var memories []models.AIMemory
	db := utils.Db.Where(
		"owner_id = ? and conversation_id = ? and vector_dimensions = ?",
		ownerID, conversationID, s.memory.VectorDimensions,
	)
	if beforeMessageID > 0 {
		db = db.Where("assistant_message_id < ?", beforeMessageID)
	}
	if err := db.Order("occurred_at desc").Limit(candidateLimit).Find(&memories).Error; err != nil {
		log.Printf("load ai memories: %v", err)
		return nil
	}

	excluded := make(map[uint]struct{}, len(history))
	for _, message := range history {
		excluded[message.ID] = struct{}{}
	}
	queryVector := BuildMemoryVector(query, s.memory.VectorDimensions)
	var scored []scoredMemory
	for _, memory := range memories {
		if _, ok := excluded[memory.UserMessageID]; ok {
			continue
		}
		if _, ok := excluded[memory.AssistantMessageID]; ok {
			continue
		}
		var vector []float64
		if err := json.Unmarshal([]byte(memory.VectorJSON), &vector); err != nil {
			continue
		}
		score := cosineSimilarity(queryVector, vector)
		if score < s.memory.MinScore {
			continue
		}
		scored = append(scored, scoredMemory{
			RetrievedMemory: RetrievedMemory{
				Content: memory.Content, OccurredAt: memory.OccurredAt, Score: score,
			},
			id: memory.ID,
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].OccurredAt.After(scored[j].OccurredAt)
		}
		return scored[i].Score > scored[j].Score
	})

	topK := s.memory.TopK
	if topK <= 0 {
		topK = 4
	}
	if len(scored) > topK {
		scored = scored[:topK]
	}
	result := make([]RetrievedMemory, len(scored))
	for i := range scored {
		result[i] = scored[i].RetrievedMemory
	}
	return result
}

func (s *Service) syncLongTermMemories(ownerID, conversationID, beforeMessageID uint) error {
	limit := s.memory.BackfillLimit
	if limit <= 0 {
		limit = 400
	}
	db := utils.Db.Where("owner_id = ? and conversation_id = ? and status = ?", ownerID, conversationID, models.AIMessageCompleted)
	if beforeMessageID > 0 {
		db = db.Where("id < ?", beforeMessageID)
	}
	var messages []models.AIMessage
	if err := db.Order("id desc").Limit(limit).Find(&messages).Error; err != nil {
		return err
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	var pairedTurns []conversationTurn
	var assistantIDs []uint
	for _, turn := range buildConversationTurns(messages, time.Now()) {
		if len(turn.messages) != 2 || turn.messages[0].Role != "user" || turn.messages[1].Role != "assistant" {
			continue
		}
		pairedTurns = append(pairedTurns, turn)
		assistantIDs = append(assistantIDs, turn.messages[1].ID)
	}
	if len(pairedTurns) == 0 {
		return nil
	}

	var indexedIDs []uint
	if err := utils.Db.Model(&models.AIMemory{}).
		Where("assistant_message_id in ? and vector_dimensions = ?", assistantIDs, s.memory.VectorDimensions).
		Pluck("assistant_message_id", &indexedIDs).Error; err != nil {
		return err
	}
	indexed := make(map[uint]struct{}, len(indexedIDs))
	for _, id := range indexedIDs {
		indexed[id] = struct{}{}
	}
	for _, turn := range pairedTurns {
		if _, exists := indexed[turn.messages[1].ID]; exists {
			continue
		}
		if err := s.storeTurnMemory(ownerID, conversationID, turn.messages[0], turn.messages[1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) storeTurnMemory(ownerID, conversationID uint, userMessage, assistantMessage models.AIMessage) error {
	if !s.memory.Enabled || userMessage.ID == 0 || assistantMessage.ID == 0 {
		return nil
	}
	content := fmt.Sprintf("用户：%s\n角色：%s", strings.TrimSpace(userMessage.Content), strings.TrimSpace(assistantMessage.Content))
	vectorJSON, err := json.Marshal(BuildMemoryVector(content, s.memory.VectorDimensions))
	if err != nil {
		return err
	}
	memory := models.AIMemory{
		OwnerID: ownerID, ConversationID: conversationID,
		UserMessageID: userMessage.ID, AssistantMessageID: assistantMessage.ID,
		Content: content, VectorJSON: string(vectorJSON), VectorDimensions: s.memory.VectorDimensions,
		OccurredAt: userMessage.CreatedAt,
	}
	return utils.Db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_message_id"}, {Name: "assistant_message_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"owner_id", "conversation_id", "content", "vector_json", "vector_dimensions", "occurred_at", "updated_at",
		}),
	}).Create(&memory).Error
}

func (s *Service) userMessageBefore(ownerID, conversationID, messageID uint) (models.AIMessage, error) {
	var message models.AIMessage
	err := utils.Db.Where(
		"owner_id = ? and conversation_id = ? and id < ? and role = ? and status = ?",
		ownerID, conversationID, messageID, "user", models.AIMessageCompleted,
	).Order("id desc").First(&message).Error
	return message, err
}

func BuildMemoryVector(text string, dimensions int) []float64 {
	if dimensions <= 0 {
		dimensions = 256
	}
	vector := make([]float64, dimensions)
	for _, token := range memoryTokens(text) {
		hasher := fnv.New32a()
		_, _ = hasher.Write([]byte(token))
		hash := hasher.Sum32()
		index := int(hash % uint32(dimensions))
		sign := 1.0
		if hash&(1<<31) != 0 {
			sign = -1
		}
		vector[index] += sign
	}
	var norm float64
	for _, value := range vector {
		norm += value * value
	}
	if norm == 0 {
		return vector
	}
	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] /= norm
	}
	return vector
}

func memoryTokens(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	var tokens []string
	var word []rune
	var semanticRunes []rune
	flushWord := func() {
		if len(word) > 0 {
			tokens = append(tokens, string(word))
			word = word[:0]
		}
	}
	for _, current := range []rune(text) {
		switch {
		case unicode.Is(unicode.Han, current):
			flushWord()
			tokens = append(tokens, string(current))
			semanticRunes = append(semanticRunes, current)
		case unicode.IsLetter(current) || unicode.IsDigit(current):
			word = append(word, current)
			semanticRunes = append(semanticRunes, current)
		default:
			flushWord()
		}
	}
	flushWord()
	for i := 0; i+1 < len(semanticRunes); i++ {
		tokens = append(tokens, string(semanticRunes[i:i+2]))
	}
	return tokens
}

func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, leftNorm, rightNorm float64
	for i := range left {
		dot += left[i] * right[i]
		leftNorm += left[i] * left[i]
		rightNorm += right[i] * right[i]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}
