package models

import (
	"fmt"
	"goChat/utils"
	"time"

	"gorm.io/gorm"
)

func Migrate() error {
	if err := normalizeAIData(); err != nil {
		return fmt.Errorf("normalize ai data: %w", err)
	}
	if err := utils.Db.AutoMigrate(
		&UserBasic{},
		&Message{},
		&Contact{},
		&GroupBasic{},
		&Community{},
		&AICharacterTemplate{},
		&AICharacter{},
		&AIConversation{},
		&AIMessage{},
		&AIUsageLog{},
		&AIMemory{},
	); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if err := seedAICharacterTemplates(); err != nil {
		return fmt.Errorf("seed ai character templates: %w", err)
	}
	return nil
}

func normalizeAIData() error {
	if !utils.Db.Migrator().HasTable(&AICharacter{}) || !utils.Db.Migrator().HasTable(&AIConversation{}) {
		return nil
	}
	// Merge historical duplicates before AutoMigrate adds the new unique indexes.
	hasMessages := utils.Db.Migrator().HasTable(&AIMessage{})
	hasUsageLogs := utils.Db.Migrator().HasTable(&AIUsageLog{})
	hasMemories := utils.Db.Migrator().HasTable(&AIMemory{})

	return utils.Db.Transaction(func(tx *gorm.DB) error {
		var characters []AICharacter
		if err := tx.Unscoped().Order("is_active desc, id asc").Find(&characters).Error; err != nil {
			return err
		}
		characterIDs := make(map[string]uint)
		for _, character := range characters {
			key := fmt.Sprintf("%d\x00%s\x00%s", character.OwnerID, character.Name, character.RelationType)
			canonicalID, exists := characterIDs[key]
			if !exists {
				characterIDs[key] = character.ID
				continue
			}
			if err := tx.Unscoped().Model(&AIConversation{}).
				Where("character_id = ?", character.ID).
				Update("character_id", canonicalID).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Delete(&character).Error; err != nil {
				return err
			}
		}

		var conversations []AIConversation
		if err := tx.Unscoped().Order("deleted_at is null desc, id asc").Find(&conversations).Error; err != nil {
			return err
		}
		type canonicalConversation struct {
			id            uint
			lastMessageAt time.Time
		}
		canonicalConversations := make(map[string]canonicalConversation)
		for _, conversation := range conversations {
			key := fmt.Sprintf("%d\x00%d", conversation.OwnerID, conversation.CharacterID)
			canonical, exists := canonicalConversations[key]
			if !exists {
				canonicalConversations[key] = canonicalConversation{
					id:            conversation.ID,
					lastMessageAt: conversation.LastMessageAt,
				}
				continue
			}
			if hasMessages {
				if err := tx.Unscoped().Model(&AIMessage{}).
					Where("conversation_id = ?", conversation.ID).
					Update("conversation_id", canonical.id).Error; err != nil {
					return err
				}
			}
			if hasUsageLogs {
				if err := tx.Unscoped().Model(&AIUsageLog{}).
					Where("conversation_id = ?", conversation.ID).
					Update("conversation_id", canonical.id).Error; err != nil {
					return err
				}
			}
			if hasMemories {
				if err := tx.Unscoped().Model(&AIMemory{}).
					Where("conversation_id = ?", conversation.ID).
					Update("conversation_id", canonical.id).Error; err != nil {
					return err
				}
			}
			canonical.lastMessageAt = latestTime(conversation.LastMessageAt, canonical.lastMessageAt)
			canonicalConversations[key] = canonical
			if err := tx.Unscoped().Model(&AIConversation{}).Where("id = ?", canonical.id).
				Update("last_message_at", canonical.lastMessageAt).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Delete(&conversation).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func latestTime(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func seedAICharacterTemplates() error {
	templates := []AICharacterTemplate{
		{Name: "女友", RelationType: "girlfriend", Description: "温柔亲密、善于倾听", DefaultPrompt: "你是用户的女友。保持温柔、亲密、尊重边界，认真倾听并给予真诚回应。", IsActive: true},
		{Name: "男友", RelationType: "boyfriend", Description: "可靠体贴、积极陪伴", DefaultPrompt: "你是用户的男友。保持可靠、体贴、尊重边界，以自然的方式陪伴和回应用户。", IsActive: true},
		{Name: "闺蜜", RelationType: "bestie", Description: "坦率热情、分享日常", DefaultPrompt: "你是用户的闺蜜。语气坦率热情，愿意分享日常、倾听烦恼并给出真诚建议。", IsActive: true},
		{Name: "兄弟", RelationType: "brother", Description: "直爽靠谱、轻松幽默", DefaultPrompt: "你是用户的好兄弟。说话直爽靠谱，可以轻松幽默，但始终尊重用户。", IsActive: true},
	}
	for _, template := range templates {
		if err := utils.Db.Where("relation_type = ?", template.RelationType).
			FirstOrCreate(&template).Error; err != nil {
			return err
		}
	}
	return nil
}
