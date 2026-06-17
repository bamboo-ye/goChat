package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"goChat/models"
	"goChat/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound         = errors.New("resource not found")
	ErrConversationBusy = errors.New("conversation is generating a response")
	ErrInvalidInput     = errors.New("invalid input")
	ErrQuotaExceeded    = errors.New("daily ai message quota exceeded")
)

type Service struct {
	provider     Provider
	historyLimit int
	maxInput     int
	maxTokens    int
	temperature  float64
	dailyLimit   int
}

func NewService(provider Provider, historyLimit, maxInput, maxTokens int, temperature float64, dailyLimit int) *Service {
	return &Service{
		provider:     provider,
		historyLimit: historyLimit,
		maxInput:     maxInput,
		maxTokens:    maxTokens,
		temperature:  temperature,
		dailyLimit:   dailyLimit,
	}
}

func NewServiceFromConfig() *Service {
	timeout := time.Duration(viper.GetInt("ai.timeoutSeconds")) * time.Second
	provider := NewOpenAICompatibleProvider(
		viper.GetString("ai.provider"),
		viper.GetString("ai.baseURL"),
		viper.GetString("ai.apiKey"),
		viper.GetString("ai.model"),
		timeout,
	)
	return NewService(
		provider,
		viper.GetInt("ai.historyLimit"),
		viper.GetInt("ai.maxInputCharacters"),
		viper.GetInt("ai.maxOutputTokens"),
		viper.GetFloat64("ai.temperature"),
		viper.GetInt("ai.dailyMessageLimit"),
	)
}

type CharacterInput struct {
	TemplateID    *uint  `json:"templateId"`
	Name          string `json:"name"`
	RelationType  string `json:"relationType"`
	Traits        string `json:"traits"`
	Hobbies       string `json:"hobbies"`
	SpeakingStyle string `json:"speakingStyle"`
	Background    string `json:"background"`
	Avatar        string `json:"avatar"`
	SystemPrompt  string `json:"systemPrompt"`
}

func (s *Service) ListTemplates() ([]models.AICharacterTemplate, error) {
	var templates []models.AICharacterTemplate
	err := utils.Db.Where("is_active = ?", true).Order("id asc").Find(&templates).Error
	return templates, err
}

func (s *Service) CreateCharacter(ownerID uint, input CharacterInput) (models.AICharacter, error) {
	if err := applyTemplateDefaults(&input); err != nil {
		return models.AICharacter{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.RelationType = strings.TrimSpace(input.RelationType)
	if !validCharacterInput(input) {
		return models.AICharacter{}, ErrInvalidInput
	}
	character := models.AICharacter{
		OwnerID: ownerID, TemplateID: input.TemplateID, Name: input.Name,
		RelationType: input.RelationType, Traits: input.Traits, Hobbies: input.Hobbies,
		SpeakingStyle: input.SpeakingStyle, Background: input.Background, Avatar: input.Avatar,
		SystemPrompt: input.SystemPrompt, IsActive: true,
	}
	if err := utils.Db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "owner_id"}, {Name: "name"}, {Name: "relation_type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"template_id", "traits", "hobbies", "speaking_style", "background", "avatar",
			"system_prompt", "is_active", "deleted_at", "updated_at",
		}),
	}).Create(&character).Error; err != nil {
		return models.AICharacter{}, err
	}
	if err := utils.Db.Where(
		"owner_id = ? and name = ? and relation_type = ?", ownerID, input.Name, input.RelationType,
	).First(&character).Error; err != nil {
		return models.AICharacter{}, err
	}
	return character, nil
}

func (s *Service) ListCharacters(ownerID uint) ([]models.AICharacter, error) {
	var characters []models.AICharacter
	err := utils.Db.Where("owner_id = ? and is_active = ?", ownerID, true).Order("updated_at desc").Find(&characters).Error
	return characters, err
}

func (s *Service) UpdateCharacter(ownerID, characterID uint, input CharacterInput) (models.AICharacter, error) {
	var character models.AICharacter
	if err := utils.Db.Where("id = ? and owner_id = ?", characterID, ownerID).First(&character).Error; err != nil {
		return models.AICharacter{}, mapNotFound(err)
	}
	if err := applyTemplateDefaults(&input); err != nil {
		return models.AICharacter{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.RelationType = strings.TrimSpace(input.RelationType)
	if !validCharacterInput(input) {
		return models.AICharacter{}, ErrInvalidInput
	}
	updates := map[string]interface{}{
		"template_id": input.TemplateID, "name": input.Name, "relation_type": input.RelationType,
		"traits": input.Traits, "hobbies": input.Hobbies, "speaking_style": input.SpeakingStyle,
		"background": input.Background, "avatar": input.Avatar, "system_prompt": input.SystemPrompt,
	}
	if err := utils.Db.Model(&character).Updates(updates).Error; err != nil {
		return models.AICharacter{}, err
	}
	if err := utils.Db.First(&character, characterID).Error; err != nil {
		return models.AICharacter{}, err
	}
	return character, nil
}

func validCharacterInput(input CharacterInput) bool {
	if input.Name == "" || input.RelationType == "" || len([]rune(input.Name)) > 64 || len([]rune(input.RelationType)) > 32 {
		return false
	}
	total := len([]rune(input.Traits + input.Hobbies + input.SpeakingStyle + input.Background + input.SystemPrompt))
	return total <= 12000
}

func applyTemplateDefaults(input *CharacterInput) error {
	if input.TemplateID == nil {
		return nil
	}
	var template models.AICharacterTemplate
	if err := utils.Db.Where("id = ? and is_active = ?", *input.TemplateID, true).First(&template).Error; err != nil {
		return mapNotFound(err)
	}
	if strings.TrimSpace(input.Name) == "" {
		input.Name = template.Name
	}
	if strings.TrimSpace(input.RelationType) == "" {
		input.RelationType = template.RelationType
	}
	if strings.TrimSpace(input.SystemPrompt) == "" {
		input.SystemPrompt = template.DefaultPrompt
	}
	if strings.TrimSpace(input.Avatar) == "" {
		input.Avatar = template.Avatar
	}
	return nil
}

func (s *Service) DeleteCharacter(ownerID, characterID uint) error {
	result := utils.Db.Model(&models.AICharacter{}).
		Where("id = ? and owner_id = ?", characterID, ownerID).
		Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) CreateConversation(ownerID, characterID uint, title string) (models.AIConversation, error) {
	var character models.AICharacter
	if err := utils.Db.Where("id = ? and owner_id = ? and is_active = ?", characterID, ownerID, true).First(&character).Error; err != nil {
		return models.AIConversation{}, mapNotFound(err)
	}
	snapshot, err := json.Marshal(CharacterSnapshot{
		Name: character.Name, RelationType: character.RelationType, Traits: character.Traits,
		Hobbies: character.Hobbies, SpeakingStyle: character.SpeakingStyle, Background: character.Background,
		Avatar: character.Avatar, SystemPrompt: character.SystemPrompt,
	})
	if err != nil {
		return models.AIConversation{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "与" + character.Name + "的聊天"
	}
	if len([]rune(title)) > 128 {
		return models.AIConversation{}, ErrInvalidInput
	}
	conversation := models.AIConversation{
		OwnerID: ownerID, CharacterID: characterID, Title: title,
		CharacterSnapshot: string(snapshot), LastMessageAt: time.Now(), Status: models.AIConversationIdle,
	}
	if err := utils.Db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "owner_id"}, {Name: "character_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"title": title, "character_snapshot": string(snapshot), "deleted_at": nil,
		}),
	}).Create(&conversation).Error; err != nil {
		return models.AIConversation{}, err
	}
	if err := utils.Db.Where("owner_id = ? and character_id = ?", ownerID, characterID).
		First(&conversation).Error; err != nil {
		return models.AIConversation{}, err
	}
	return conversation, nil
}

func (s *Service) ListConversations(ownerID uint) ([]models.AIConversation, error) {
	var conversations []models.AIConversation
	err := utils.Db.Where("owner_id = ?", ownerID).Order("last_message_at desc").Find(&conversations).Error
	return conversations, err
}

func (s *Service) DeleteConversation(ownerID, conversationID uint) error {
	result := utils.Db.Where("id = ? and owner_id = ?", conversationID, ownerID).Delete(&models.AIConversation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ListMessages(ownerID, conversationID uint, limit, offset int) ([]models.AIMessage, error) {
	if err := s.ensureConversationOwner(ownerID, conversationID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var messages []models.AIMessage
	err := utils.Db.Where("conversation_id = ? and owner_id = ?", conversationID, ownerID).
		Order("id desc").Limit(limit).Offset(offset).Find(&messages).Error
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, err
}

func (s *Service) SendMessage(ctx context.Context, ownerID, conversationID uint, content string) (models.AIMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" || (s.maxInput > 0 && len([]rune(content)) > s.maxInput) {
		return models.AIMessage{}, ErrInvalidInput
	}
	if err := s.checkDailyQuota(ownerID); err != nil {
		return models.AIMessage{}, err
	}
	conversation, err := s.lockConversation(ownerID, conversationID)
	if err != nil {
		return models.AIMessage{}, err
	}
	defer utils.Db.Model(&models.AIConversation{}).Where("id = ?", conversationID).
		Update("status", models.AIConversationIdle)

	userMessage := models.AIMessage{
		ConversationID: conversationID, OwnerID: ownerID, Role: "user",
		Content: content, Status: models.AIMessageCompleted,
	}
	assistantMessage := models.AIMessage{
		ConversationID: conversationID, OwnerID: ownerID, Role: "assistant",
		Content: "", Status: models.AIMessagePending,
	}
	if err := utils.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&userMessage).Error; err != nil {
			return err
		}
		if err := tx.Create(&assistantMessage).Error; err != nil {
			return err
		}
		return tx.Model(&models.AIConversation{}).Where("id = ?", conversationID).
			Update("last_message_at", time.Now()).Error
	}); err != nil {
		return models.AIMessage{}, err
	}

	var snapshot CharacterSnapshot
	if err := json.Unmarshal([]byte(conversation.CharacterSnapshot), &snapshot); err != nil {
		return s.failMessage(assistantMessage, err)
	}
	history, err := s.recentHistory(conversationID)
	if err != nil {
		return s.failMessage(assistantMessage, err)
	}

	startedAt := time.Now()
	result, callErr := s.provider.Complete(ctx, CompletionRequest{
		Messages: BuildPrompt(snapshot, history), Temperature: s.temperature, MaxTokens: s.maxTokens,
	})
	usage := models.AIUsageLog{
		OwnerID: ownerID, ConversationID: conversationID, MessageID: assistantMessage.ID,
		Provider: s.provider.Name(), ModelName: result.Model, PromptTokens: result.PromptTokens,
		CompletionTokens: result.CompletionTokens, LatencyMS: time.Since(startedAt).Milliseconds(),
		Status: models.AIMessageCompleted,
	}
	if callErr != nil {
		usage.Status = models.AIMessageFailed
		usage.ErrorMessage = callErr.Error()
		_ = utils.Db.Create(&usage).Error
		return s.failMessage(assistantMessage, callErr)
	}

	assistantMessage.Content = result.Content
	assistantMessage.Status = models.AIMessageCompleted
	assistantMessage.ModelName = result.Model
	assistantMessage.PromptTokens = result.PromptTokens
	assistantMessage.CompletionTokens = result.CompletionTokens
	if err := utils.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&assistantMessage).Error; err != nil {
			return err
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		return tx.Model(&models.AIConversation{}).Where("id = ?", conversationID).
			Update("last_message_at", time.Now()).Error
	}); err != nil {
		return models.AIMessage{}, err
	}
	return assistantMessage, nil
}

func (s *Service) RetryMessage(ctx context.Context, ownerID, conversationID, messageID uint) (models.AIMessage, error) {
	if err := s.checkDailyQuota(ownerID); err != nil {
		return models.AIMessage{}, err
	}
	conversation, err := s.lockConversation(ownerID, conversationID)
	if err != nil {
		return models.AIMessage{}, err
	}
	defer utils.Db.Model(&models.AIConversation{}).Where("id = ?", conversationID).
		Update("status", models.AIConversationIdle)

	var assistantMessage models.AIMessage
	if err := utils.Db.Where(
		"id = ? and conversation_id = ? and owner_id = ? and role = ? and status = ?",
		messageID, conversationID, ownerID, "assistant", models.AIMessageFailed,
	).First(&assistantMessage).Error; err != nil {
		return models.AIMessage{}, mapNotFound(err)
	}
	if err := utils.Db.Model(&assistantMessage).Updates(map[string]interface{}{
		"status": models.AIMessagePending, "error_message": "",
	}).Error; err != nil {
		return models.AIMessage{}, err
	}

	var snapshot CharacterSnapshot
	if err := json.Unmarshal([]byte(conversation.CharacterSnapshot), &snapshot); err != nil {
		return s.failMessage(assistantMessage, err)
	}
	history, err := s.recentHistoryBefore(conversationID, messageID)
	if err != nil {
		return s.failMessage(assistantMessage, err)
	}
	return s.completeAssistantMessage(ctx, ownerID, conversationID, assistantMessage, snapshot, history)
}

func (s *Service) completeAssistantMessage(ctx context.Context, ownerID, conversationID uint, assistantMessage models.AIMessage, snapshot CharacterSnapshot, history []models.AIMessage) (models.AIMessage, error) {
	startedAt := time.Now()
	result, callErr := s.provider.Complete(ctx, CompletionRequest{
		Messages: BuildPrompt(snapshot, history), Temperature: s.temperature, MaxTokens: s.maxTokens,
	})
	usage := models.AIUsageLog{
		OwnerID: ownerID, ConversationID: conversationID, MessageID: assistantMessage.ID,
		Provider: s.provider.Name(), ModelName: result.Model, PromptTokens: result.PromptTokens,
		CompletionTokens: result.CompletionTokens, LatencyMS: time.Since(startedAt).Milliseconds(),
		Status: models.AIMessageCompleted,
	}
	if callErr != nil {
		usage.Status = models.AIMessageFailed
		usage.ErrorMessage = callErr.Error()
		_ = utils.Db.Create(&usage).Error
		return s.failMessage(assistantMessage, callErr)
	}

	assistantMessage.Content = result.Content
	assistantMessage.Status = models.AIMessageCompleted
	assistantMessage.ModelName = result.Model
	assistantMessage.PromptTokens = result.PromptTokens
	assistantMessage.CompletionTokens = result.CompletionTokens
	assistantMessage.ErrorMessage = ""
	if err := utils.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&assistantMessage).Error; err != nil {
			return err
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		return tx.Model(&models.AIConversation{}).Where("id = ?", conversationID).
			Update("last_message_at", time.Now()).Error
	}); err != nil {
		return models.AIMessage{}, err
	}
	return assistantMessage, nil
}

func (s *Service) checkDailyQuota(ownerID uint) error {
	if s.dailyLimit <= 0 {
		return nil
	}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var count int64
	if err := utils.Db.Model(&models.AIUsageLog{}).
		Where("owner_id = ? and created_at >= ?", ownerID, startOfDay).Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(s.dailyLimit) {
		return ErrQuotaExceeded
	}
	return nil
}

func (s *Service) lockConversation(ownerID, conversationID uint) (models.AIConversation, error) {
	var conversation models.AIConversation
	if err := utils.Db.Where("id = ? and owner_id = ?", conversationID, ownerID).First(&conversation).Error; err != nil {
		return conversation, mapNotFound(err)
	}
	result := utils.Db.Model(&models.AIConversation{}).
		Where("id = ? and owner_id = ? and status <> ?", conversationID, ownerID, models.AIConversationGenerating).
		Update("status", models.AIConversationGenerating)
	if result.Error != nil {
		return conversation, result.Error
	}
	if result.RowsAffected == 0 {
		return conversation, ErrConversationBusy
	}
	return conversation, nil
}

func (s *Service) recentHistory(conversationID uint) ([]models.AIMessage, error) {
	limit := s.historyLimit
	if limit <= 0 {
		limit = 20
	}
	var messages []models.AIMessage
	err := utils.Db.Where("conversation_id = ? and status = ?", conversationID, models.AIMessageCompleted).
		Order("id desc").Limit(limit).Find(&messages).Error
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, err
}

func (s *Service) recentHistoryBefore(conversationID, messageID uint) ([]models.AIMessage, error) {
	limit := s.historyLimit
	if limit <= 0 {
		limit = 20
	}
	var messages []models.AIMessage
	err := utils.Db.Where("conversation_id = ? and id < ? and status = ?", conversationID, messageID, models.AIMessageCompleted).
		Order("id desc").Limit(limit).Find(&messages).Error
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, err
}

func (s *Service) failMessage(message models.AIMessage, cause error) (models.AIMessage, error) {
	message.Status = models.AIMessageFailed
	message.ErrorMessage = cause.Error()
	_ = utils.Db.Save(&message).Error
	return message, cause
}

func (s *Service) ensureConversationOwner(ownerID, conversationID uint) error {
	var count int64
	if err := utils.Db.Model(&models.AIConversation{}).
		Where("id = ? and owner_id = ?", conversationID, ownerID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("database error: %w", err)
}
