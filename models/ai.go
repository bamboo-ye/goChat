package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	AIConversationIdle       = "idle"
	AIConversationGenerating = "generating"
	AIMessagePending         = "pending"
	AIMessageCompleted       = "completed"
	AIMessageFailed          = "failed"
)

type AICharacterTemplate struct {
	gorm.Model
	Name          string `json:"name" gorm:"size:64;not null"`
	RelationType  string `json:"relationType" gorm:"size:32;not null;uniqueIndex"`
	Description   string `json:"description" gorm:"size:255"`
	DefaultPrompt string `json:"defaultPrompt" gorm:"type:text;not null"`
	Avatar        string `json:"avatar" gorm:"size:512"`
	IsActive      bool   `json:"isActive" gorm:"not null;default:true"`
}

func (AICharacterTemplate) TableName() string {
	return "ai_character_template"
}

type AICharacter struct {
	gorm.Model
	OwnerID       uint   `json:"-" gorm:"not null;index;uniqueIndex:uk_ai_character_owner_name_relation,priority:1"`
	TemplateID    *uint  `json:"templateId,omitempty" gorm:"index"`
	Name          string `json:"name" gorm:"size:64;not null;uniqueIndex:uk_ai_character_owner_name_relation,priority:2"`
	RelationType  string `json:"relationType" gorm:"size:32;not null;index;uniqueIndex:uk_ai_character_owner_name_relation,priority:3"`
	Traits        string `json:"traits" gorm:"type:text"`
	Hobbies       string `json:"hobbies" gorm:"type:text"`
	SpeakingStyle string `json:"speakingStyle" gorm:"type:text"`
	Background    string `json:"background" gorm:"type:text"`
	Avatar        string `json:"avatar" gorm:"size:512"`
	SystemPrompt  string `json:"systemPrompt" gorm:"type:text"`
	IsActive      bool   `json:"isActive" gorm:"not null;default:true"`
}

func (AICharacter) TableName() string {
	return "ai_character"
}

type AIConversation struct {
	gorm.Model
	OwnerID           uint      `json:"-" gorm:"not null;index:idx_ai_conversation_owner_updated;uniqueIndex:uk_ai_conversation_owner_character,priority:1"`
	CharacterID       uint      `json:"characterId" gorm:"not null;index;uniqueIndex:uk_ai_conversation_owner_character,priority:2"`
	Title             string    `json:"title" gorm:"size:128;not null"`
	CharacterSnapshot string    `json:"characterSnapshot" gorm:"type:text;not null"`
	LastMessageAt     time.Time `json:"lastMessageAt" gorm:"index:idx_ai_conversation_owner_updated"`
	Status            string    `json:"status" gorm:"size:24;not null;default:idle"`
}

func (AIConversation) TableName() string {
	return "ai_conversation"
}

type AIMessage struct {
	gorm.Model
	ConversationID   uint   `json:"conversationId" gorm:"not null;index:idx_ai_message_conversation_created"`
	OwnerID          uint   `json:"-" gorm:"not null;index"`
	Role             string `json:"role" gorm:"size:16;not null"`
	Content          string `json:"content" gorm:"type:text;not null"`
	Status           string `json:"status" gorm:"size:24;not null"`
	ModelName        string `json:"model,omitempty" gorm:"column:model;size:128"`
	PromptTokens     int    `json:"promptTokens,omitempty"`
	CompletionTokens int    `json:"completionTokens,omitempty"`
	ErrorMessage     string `json:"-" gorm:"type:text"`
}

func (AIMessage) TableName() string {
	return "ai_message"
}

type AIUsageLog struct {
	gorm.Model
	OwnerID          uint   `json:"-" gorm:"not null;index"`
	ConversationID   uint   `json:"conversationId" gorm:"not null;index"`
	MessageID        uint   `json:"messageId" gorm:"not null;index"`
	Provider         string `json:"provider" gorm:"size:64;not null"`
	ModelName        string `json:"model" gorm:"column:model;size:128;not null"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	LatencyMS        int64  `json:"latencyMs"`
	Status           string `json:"status" gorm:"size:24;not null"`
	ErrorMessage     string `json:"errorMessage,omitempty" gorm:"type:text"`
}

func (AIUsageLog) TableName() string {
	return "ai_usage_log"
}
